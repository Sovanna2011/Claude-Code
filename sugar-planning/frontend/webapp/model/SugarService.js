sap.ui.define([
	"sap/ui/base/Object",
	"sap/ui/model/json/JSONModel"
], function (BaseObject, JSONModel) {
	"use strict";

	/**
	 * SugarService is the single place that talks to the API.
	 *
	 * Keeping every request here means the access token, the correlation id and
	 * the problem-details handling are applied once rather than in every
	 * controller, and a controller never has to know a URL.
	 */
	return BaseObject.extend("sugarplan.model.SugarService", {

		constructor: function (sBaseUrl) {
			BaseObject.call(this);
			this._sBase = sBaseUrl || "/api/v1";
			this._sToken = window.sessionStorage.getItem("sugarplan.token") || "";
			this._oSession = null;
		},

		/** The bearer token is held in session storage so a page reload keeps
		 * the session, and closing the tab ends it. */
		setToken: function (sToken) {
			this._sToken = sToken || "";
			if (sToken) {
				window.sessionStorage.setItem("sugarplan.token", sToken);
			} else {
				window.sessionStorage.removeItem("sugarplan.token");
				this._oSession = null;
			}
		},

		hasToken: function () {
			return !!this._sToken;
		},

		getSessionProfile: function () {
			return this._oSession;
		},

		/**
		 * request performs one API call.
		 *
		 * Errors are rejected with the parsed problem document, so a caller can
		 * show `detail` to the user and bind `errors[]` to a MessagePopover
		 * without re-parsing anything.
		 */
		request: function (sMethod, sPath, oOptions) {
			var oOpts = oOptions || {};
			var sUrl = this._sBase + sPath;
			var oHeaders = { "Accept": "application/json" };

			if (this._sToken) {
				oHeaders["Authorization"] = "Bearer " + this._sToken;
			}
			if (oOpts.body !== undefined) {
				oHeaders["Content-Type"] = "application/json";
			}
			// A raw body is the file itself - an import upload - rather than a
			// JSON document, so it is sent unchanged with its own content type.
			if (oOpts.rawBody !== undefined) {
				oHeaders["Content-Type"] = oOpts.contentType || "application/octet-stream";
			}
			if (oOpts.etag) {
				oHeaders["If-Match"] = oOpts.etag;
			}
			if (oOpts.idempotencyKey) {
				oHeaders["Idempotency-Key"] = oOpts.idempotencyKey;
			}
			Object.assign(oHeaders, oOpts.headers || {});

			return fetch(sUrl, {
				method: sMethod,
				headers: oHeaders,
				body: oOpts.rawBody !== undefined ? oOpts.rawBody
					: (oOpts.body !== undefined ? JSON.stringify(oOpts.body) : undefined),
				credentials: "same-origin"
			}).then(function (oResponse) {
				var sEtag = oResponse.headers.get("ETag");

				if (oOpts.raw) {
					if (!oResponse.ok) {
						return oResponse.json().then(function (oProblem) {
							throw oProblem;
						});
					}
					return oResponse.blob();
				}

				if (oResponse.status === 204) {
					return { etag: sEtag };
				}

				return oResponse.text().then(function (sText) {
					var oBody = null;
					if (sText) {
						try {
							oBody = JSON.parse(sText);
						} catch (e) {
							oBody = { detail: sText };
						}
					}
					if (!oResponse.ok) {
						throw Object.assign({
							status: oResponse.status,
							title: "The request failed",
							detail: ""
						}, oBody || {});
					}
					if (oBody && typeof oBody === "object" && sEtag) {
						Object.defineProperty(oBody, "__etag", {
							value: sEtag, enumerable: false, writable: true
						});
					}
					return oBody;
				});
			});
		},

		get: function (sPath) {
			return this.request("GET", sPath);
		},

		post: function (sPath, oBody, oOptions) {
			return this.request("POST", sPath, Object.assign({ body: oBody }, oOptions || {}));
		},

		put: function (sPath, oBody, oOptions) {
			return this.request("PUT", sPath, Object.assign({ body: oBody }, oOptions || {}));
		},

		del: function (sPath, oOptions) {
			return this.request("DELETE", sPath, oOptions || {});
		},

		// ------------------------------------------------------------------
		// Session
		// ------------------------------------------------------------------

		/** devLogin is available only when the API runs in development mode. */
		devLogin: function (sUsername) {
			var that = this;
			return this.post("/auth/dev-login", { username: sUsername }).then(function (oResult) {
				that.setToken(oResult.accessToken);
				that._oSession = oResult.profile;
				return oResult.profile;
			});
		},

		loadSession: function () {
			var that = this;
			return this.get("/session").then(function (oProfile) {
				that._oSession = oProfile;
				return oProfile;
			});
		},

		logout: function () {
			this.setToken("");
		},

		/** can reports whether the signed-in user holds a permission, so the UI
		 * can hide an action the backend would refuse anyway. */
		can: function (sPermission) {
			var oSession = this._oSession;
			return !!(oSession && oSession.permissions && oSession.permissions.indexOf(sPermission) >= 0);
		},

		// ------------------------------------------------------------------
		// Domain calls
		// ------------------------------------------------------------------

		listSeasons: function () {
			return this.get("/seasons?$top=100");
		},

		listVersions: function (sSeasonId) {
			return this.get("/seasons/" + encodeURIComponent(sSeasonId) + "/versions?$top=100");
		},

		getVersion: function (sVersionId) {
			return this.get("/versions/" + encodeURIComponent(sVersionId));
		},

		dashboard: function (sSeasonId, sVersionId) {
			var sPath = "/dashboard?seasonId=" + encodeURIComponent(sSeasonId);
			if (sVersionId) {
				sPath += "&versionId=" + encodeURIComponent(sVersionId);
			}
			return this.get(sPath);
		},

		generate: function (sVersionId, bReplace) {
			return this.post("/versions/" + encodeURIComponent(sVersionId) + "/generate",
				{ replace: !!bReplace });
		},

		transition: function (sVersionId, oPayload, sEtag) {
			return this.post("/versions/" + encodeURIComponent(sVersionId) + "/transition",
				oPayload, { etag: sEtag });
		},

		copyVersion: function (sVersionId, oPayload) {
			return this.post("/versions/" + encodeURIComponent(sVersionId) + "/copy", oPayload);
		},

		compare: function (oPayload) {
			return this.post("/versions/compare", oPayload);
		},

		saveAssumption: function (sVersionId, oAssumption) {
			return this.put("/versions/" + encodeURIComponent(sVersionId) + "/assumptions", oAssumption);
		},

		listRows: function (sVersionId, sKind, oParams) {
			var aQuery = [];
			Object.keys(oParams || {}).forEach(function (sKey) {
				if (oParams[sKey]) {
					aQuery.push(encodeURIComponent(sKey) + "=" + encodeURIComponent(oParams[sKey]));
				}
			});
			var sQuery = aQuery.length ? "?" + aQuery.join("&") : "";
			return this.get("/versions/" + encodeURIComponent(sVersionId) + "/" + sKind + sQuery);
		},

		saveRows: function (sVersionId, sKind, aRows, bPartial) {
			return this.post("/versions/" + encodeURIComponent(sVersionId) + "/" + sKind,
				{ rows: aRows, partial: !!bPartial });
		},

		materialRequirements: function (sVersionId) {
			return this.get("/versions/" + encodeURIComponent(sVersionId) + "/material-requirements");
		},

		listReports: function () {
			return this.get("/reports");
		},

		runReport: function (sCode, oParams) {
			var aQuery = [];
			Object.keys(oParams || {}).forEach(function (sKey) {
				if (oParams[sKey]) {
					aQuery.push(encodeURIComponent(sKey) + "=" + encodeURIComponent(oParams[sKey]));
				}
			});
			return this.get("/reports/" + encodeURIComponent(sCode) + "?" + aQuery.join("&"));
		},

		/** downloadCertificate fetches the certificate of analysis for a sample
		 * and hands it to the browser. The PDF is what goes in the envelope with
		 * a consignment, so it is named after the sample rather than after a
		 * report code. */
		downloadCertificate: function (sSampleId, sSampleNo, sFormat) {
			var sPath = "/quality/samples/" + encodeURIComponent(sSampleId) +
				"/certificate?format=" + encodeURIComponent(sFormat);
			return this.request("GET", sPath, { raw: true }).then(function (oBlob) {
				this._save(oBlob, "COA-" + sSampleNo + "." + sFormat);
			}.bind(this));
		},

		/** _save hands a blob to the browser as a download. */
		_save: function (oBlob, sName) {
			var sUrl = window.URL.createObjectURL(oBlob);
			var oLink = document.createElement("a");
			oLink.href = sUrl;
			oLink.download = sName;
			document.body.appendChild(oLink);
			oLink.click();
			document.body.removeChild(oLink);
			window.URL.revokeObjectURL(sUrl);
		},

		/** downloadReport fetches an export and hands it to the browser. */
		downloadReport: function (sCode, sFormat, oParams) {
			var aQuery = ["format=" + encodeURIComponent(sFormat)];
			Object.keys(oParams || {}).forEach(function (sKey) {
				if (oParams[sKey]) {
					aQuery.push(encodeURIComponent(sKey) + "=" + encodeURIComponent(oParams[sKey]));
				}
			});
			var sPath = "/reports/" + encodeURIComponent(sCode) + "?" + aQuery.join("&");
			return this.request("GET", sPath, { raw: true }).then(function (oBlob) {
				this._save(oBlob, sCode + "." + sFormat);
			}.bind(this));
		},

		listMaster: function (sEntity, oParams) {
			var aQuery = ["$top=200"];
			Object.keys(oParams || {}).forEach(function (sKey) {
				if (oParams[sKey]) {
					aQuery.push(encodeURIComponent(sKey) + "=" + encodeURIComponent(oParams[sKey]));
				}
			});
			return this.get("/master/" + sEntity + "?" + aQuery.join("&"));
		},

		// ------------------------------------------------------------------
		// Execution: stock, postings, orders, quality, maintenance
		// ------------------------------------------------------------------

		/** query builds a query string from the parameters that have a value,
		 * so a caller never has to think about the first "?" or a stray "&". */
		query: function (oParams) {
			var aQuery = [];
			Object.keys(oParams || {}).forEach(function (sKey) {
				var vValue = oParams[sKey];
				if (vValue !== undefined && vValue !== null && vValue !== "") {
					aQuery.push(encodeURIComponent(sKey) + "=" + encodeURIComponent(vValue));
				}
			});
			return aQuery.length ? "?" + aQuery.join("&") : "";
		},

		listStock: function (oParams) {
			return this.get("/stock" + this.query(oParams));
		},

		listDocuments: function (oParams) {
			return this.get("/inventory/documents" + this.query(oParams));
		},

		/**
		 * postDocument moves stock.
		 *
		 * Every posting carries an idempotency key. A warehouse keeper on a bad
		 * connection who presses the button twice, or a browser that retries the
		 * request itself, must not move the balance twice; the key means the
		 * second attempt replays the first document instead.
		 */
		postDocument: function (oPosting) {
			return this.post("/inventory/documents", oPosting,
				{ idempotencyKey: this.newIdempotencyKey() });
		},

		reverseDocument: function (sDocumentId, oPayload) {
			return this.post("/inventory/documents/" + encodeURIComponent(sDocumentId) + "/reverse",
				oPayload, { idempotencyKey: this.newIdempotencyKey() });
		},

		listOrders: function (oParams) {
			return this.get("/production-orders" + this.query(oParams));
		},

		getOrder: function (sOrderId) {
			return this.get("/production-orders/" + encodeURIComponent(sOrderId));
		},

		createOrder: function (oOrder) {
			return this.post("/production-orders", oOrder);
		},

		ordersFromPlan: function (sVersionId, oPayload) {
			return this.post("/versions/" + encodeURIComponent(sVersionId) + "/production-orders",
				oPayload, { idempotencyKey: this.newIdempotencyKey() });
		},

		orderAction: function (sOrderId, oPayload, sEtag) {
			return this.post("/production-orders/" + encodeURIComponent(sOrderId) + "/action",
				oPayload, { etag: sEtag });
		},

		confirmOrder: function (sOrderId, oPayload) {
			return this.post("/production-orders/" + encodeURIComponent(sOrderId) + "/confirm",
				oPayload, { idempotencyKey: this.newIdempotencyKey() });
		},

		reverseConfirmation: function (sConfirmationId, oPayload) {
			return this.post("/confirmations/" + encodeURIComponent(sConfirmationId) + "/reverse",
				oPayload, { idempotencyKey: this.newIdempotencyKey() });
		},

		listQualityParameters: function () {
			return this.get("/quality/parameters");
		},

		listQualitySpecs: function (sProductId, sOn) {
			return this.get("/quality/specs" + this.query({ productId: sProductId, on: sOn }));
		},

		listSamples: function (oParams) {
			return this.get("/quality/samples" + this.query(oParams));
		},

		getSample: function (sSampleId) {
			return this.get("/quality/samples/" + encodeURIComponent(sSampleId));
		},

		createSample: function (oSample) {
			return this.post("/quality/samples", oSample);
		},

		recordResults: function (sSampleId, oPayload) {
			return this.post("/quality/samples/" + encodeURIComponent(sSampleId) + "/results", oPayload);
		},

		listHolds: function (oParams) {
			return this.get("/quality/holds" + this.query(oParams));
		},

		releaseHold: function (sHoldId, oPayload, sEtag) {
			return this.post("/quality/holds/" + encodeURIComponent(sHoldId) + "/release",
				oPayload, { etag: sEtag });
		},

		listDowntime: function (oParams) {
			return this.get("/downtime" + this.query(oParams));
		},

		saveDowntime: function (oEvent) {
			return this.post("/downtime", oEvent);
		},

		listMaintenance: function (oParams) {
			return this.get("/maintenance" + this.query(oParams));
		},

		saveMaintenance: function (oWindow, sEtag) {
			return this.put("/maintenance", oWindow, { etag: sEtag });
		},

		// ------------------------------------------------------------------
		// Costing
		// ------------------------------------------------------------------

		listCostElements: function () {
			return this.get("/costing/elements?$top=200");
		},

		saveCostElement: function (oElement) {
			return this.put("/costing/elements", oElement);
		},

		listCostRates: function (oParams) {
			return this.get("/costing/rates" + this.query(oParams));
		},

		saveCostRate: function (oRate) {
			return this.put("/costing/rates", oRate);
		},

		deleteCostRate: function (sId) {
			return this.del("/costing/rates/" + encodeURIComponent(sId));
		},

		listExchangeRates: function () {
			return this.get("/costing/exchange-rates");
		},

		saveExchangeRate: function (oRate) {
			return this.put("/costing/exchange-rates", oRate);
		},

		/** costRun prices a period. Saving one is a write and carries a key;
		 * a run that is only being looked at writes nothing. */
		costRun: function (oRequest) {
			var oOptions = oRequest.save ? { idempotencyKey: this.newIdempotencyKey() } : {};
			return this.post("/costing/runs", oRequest, oOptions);
		},

		listCostRuns: function (oParams) {
			return this.get("/costing/runs" + this.query(oParams));
		},

		getCostRun: function (sId) {
			return this.get("/costing/runs/" + encodeURIComponent(sId));
		},

		/** newIdempotencyKey returns a key unique to one user action. */
		newIdempotencyKey: function () {
			if (window.crypto && window.crypto.randomUUID) {
				return window.crypto.randomUUID();
			}
			return "k-" + Date.now() + "-" + Math.random().toString(36).slice(2, 10);
		},

		listAudit: function (oParams) {
			var aQuery = ["$top=200"];
			Object.keys(oParams || {}).forEach(function (sKey) {
				if (oParams[sKey]) {
					aQuery.push(encodeURIComponent(sKey) + "=" + encodeURIComponent(oParams[sKey]));
				}
			});
			return this.get("/audit?" + aQuery.join("&"));
		},

		// ------------------------------------------------------------------
		// Inbox
		// ------------------------------------------------------------------

		/** listNotifications reads the caller's inbox. The unread count comes
		 * back with it so the badge cannot disagree with the list. */
		listNotifications: function (bUnreadOnly) {
			return this.get("/notifications?$top=100" + (bUnreadOnly ? "&unreadOnly=true" : ""));
		},

		markNotificationRead: function (sId) {
			return this.post("/notifications/" + encodeURIComponent(sId) + "/read", null);
		},

		// ------------------------------------------------------------------
		// Imports
		// ------------------------------------------------------------------

		/** listImportMappings returns the templates a site has set up. */
		listImportMappings: function (sKind) {
			return this.get("/import-mappings" + (sKind ? "?kind=" + encodeURIComponent(sKind) : ""));
		},

		/**
		 * stageImport uploads a file.
		 *
		 * The body is the file itself, so nothing here builds a multipart form:
		 * the mapping, the version and the series are query parameters, and the
		 * browser sends the bytes it read from disk unchanged.
		 */
		stageImport: function (oFile, oParams) {
			var aQuery = [];
			Object.keys(oParams || {}).forEach(function (sKey) {
				if (oParams[sKey]) {
					aQuery.push(encodeURIComponent(sKey) + "=" + encodeURIComponent(oParams[sKey]));
				}
			});
			aQuery.push("fileName=" + encodeURIComponent(oFile.name));
			return this.request("POST", "/imports?" + aQuery.join("&"), {
				rawBody: oFile,
				contentType: oFile.type || "application/octet-stream"
			});
		},

		/** getImport reads a staged job back, optionally only its failed rows. */
		getImport: function (sId, bErrorsOnly) {
			return this.get("/imports/" + encodeURIComponent(sId) +
				"?$top=500" + (bErrorsOnly ? "&errorsOnly=true" : ""));
		},

		listImports: function () {
			return this.get("/imports?$top=50");
		},

		commitImport: function (sId, bPartial) {
			return this.post("/imports/" + encodeURIComponent(sId) + "/commit",
				{ partial: !!bPartial });
		},

		cancelImport: function (sId) {
			return this.post("/imports/" + encodeURIComponent(sId) + "/cancel", null);
		},

		/** downloadImportErrors fetches the row-level error file. */
		downloadImportErrors: function (sId, sFileName) {
			return this.request("GET", "/imports/" + encodeURIComponent(sId) + "/errors",
				{ raw: true }).then(function (oBlob) {
					// The uploaded name already carries an extension; a second
					// one would give "cane.csv.csv".
					var sBase = (sFileName || sId).replace(/\.[^.]+$/, "");
					this._save(oBlob, "errors-" + sBase + ".csv");
				}.bind(this));
		},

		// ------------------------------------------------------------------
		// Interfaces
		// ------------------------------------------------------------------

		/** listEvents reads the outbox: what this system has to tell the
		 * connected systems, and what it has not managed to tell them yet. */
		listEvents: function (oParams) {
			var aQuery = ["$top=200"];
			Object.keys(oParams || {}).forEach(function (sKey) {
				if (oParams[sKey]) {
					aQuery.push(encodeURIComponent(sKey) + "=" + encodeURIComponent(oParams[sKey]));
				}
			});
			return this.get("/integration/events?" + aQuery.join("&"));
		},

		/** retryEvent delivers one event now, whatever its backoff says. */
		retryEvent: function (sId) {
			return this.post("/integration/events/" + encodeURIComponent(sId) + "/retry", null);
		},

		/** dispatch runs a dispatcher pass on demand, which is what drains a
		 * backlog after an outage rather than waiting for the next tick. */
		dispatch: function () {
			return this.post("/integration/dispatch", null);
		},

		/** newModel wraps a payload in a JSONModel with a generous size limit,
		 * because a season is 137 days across several dimensions. */
		// ------------------------------------------------------------------
		// Saved views (variant management)
		// ------------------------------------------------------------------

		listViews: function (sPage) {
			return this.get("/views" + this.query({ page: sPage }));
		},

		saveView: function (oView) {
			return this.put("/views", oView);
		},

		deleteView: function (sId) {
			return this.del("/views/" + encodeURIComponent(sId));
		},

		setDefaultView: function (sId, bOn) {
			return this.post("/views/" + encodeURIComponent(sId) + "/default",
				{ "default": bOn !== false });
		},

		newModel: function (oData) {
			var oModel = new JSONModel(oData);
			oModel.setSizeLimit(100000);
			return oModel;
		}
	});
});
