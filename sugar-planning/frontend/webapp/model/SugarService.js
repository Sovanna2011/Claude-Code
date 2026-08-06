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
				body: oOpts.body !== undefined ? JSON.stringify(oOpts.body) : undefined,
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
				var sUrl = window.URL.createObjectURL(oBlob);
				var oLink = document.createElement("a");
				oLink.href = sUrl;
				oLink.download = sCode + "." + sFormat;
				document.body.appendChild(oLink);
				oLink.click();
				document.body.removeChild(oLink);
				window.URL.revokeObjectURL(sUrl);
			});
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

		listAudit: function (oParams) {
			var aQuery = ["$top=200"];
			Object.keys(oParams || {}).forEach(function (sKey) {
				if (oParams[sKey]) {
					aQuery.push(encodeURIComponent(sKey) + "=" + encodeURIComponent(oParams[sKey]));
				}
			});
			return this.get("/audit?" + aQuery.join("&"));
		},

		/** newModel wraps a payload in a JSONModel with a generous size limit,
		 * because a season is 137 days across several dimensions. */
		newModel: function (oData) {
			var oModel = new JSONModel(oData);
			oModel.setSizeLimit(100000);
			return oModel;
		}
	});
});
