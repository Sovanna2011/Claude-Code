sap.ui.define([
	"sugarplan/controller/BaseController"
], function (BaseController) {
	"use strict";

	/**
	 * PlanningBoard is the daily grid. It is the screen that replaces the
	 * worksheet, so three things matter most: an edit must be obviously an edit,
	 * an unsaved change must never be lost silently, and a rejected save must
	 * name the row and the field it objected to.
	 */
	return BaseController.extend("sugarplan.controller.PlanningBoard", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({
				version: {}, kind: "cane", from: "", to: "", series: "PLAN",
				dirty: false, readOnly: true, readOnlyReason: "", rowCount: 0
			}), "view");
			this.getView().setModel(this.getService().newModel({
				cane: [], production: [], storage: [], shipments: []
			}), "rows");

			this.getRouter().getRoute("board").attachPatternMatched(this._onDisplay, this);

			// Unsaved-change protection for a browser reload or tab close. The
			// in-app navigation guard is in onNavBack.
			this._fnBeforeUnload = function (oEvent) {
				if (this.getView().getModel("view").getProperty("/dirty")) {
					oEvent.preventDefault();
					oEvent.returnValue = "";
				}
			}.bind(this);
			window.addEventListener("beforeunload", this._fnBeforeUnload);
		},

		onExit: function () {
			window.removeEventListener("beforeunload", this._fnBeforeUnload);
		},

		_onDisplay: function (oEvent) {
			var oArgs = oEvent.getParameter("arguments") || {};
			this._sVersionId = oArgs.versionId;
			this._dirtyKeys = {};

			// A drill-down from a KPI arrives with the grid it meant: which set
			// of rows, which series, and the fortnight it was describing. Landing
			// on the season's first fortnight instead would make the reader
			// reproduce a filter they had already expressed by clicking.
			var oQuery = oArgs["?query"] || {};
			var oView = this.getView().getModel("view");
			["kind", "series", "from", "to"].forEach(function (sField) {
				if (oQuery[sField]) {
					oView.setProperty("/" + sField, oQuery[sField]);
				}
			});

			var that = this;
			this.setBusy(true);

			Promise.all([
				this.getService().getVersion(this._sVersionId),
				this.getService().listMaster("products"),
				this.getService().listMaster("warehouses"),
				this.getService().listMaster("shipment-channels")
			]).then(function (aResults) {
				var oDetail = aResults[0];
				that._mProducts = that._index(aResults[1].value);
				that._mWarehouses = that._index(aResults[2].value);
				that._mChannels = that._index(aResults[3].value);

				var oVersion = oDetail.version;
				var oView = that.getView().getModel("view");
				oView.setProperty("/version", oVersion);

				// Default the range to the first fortnight of the season, which
				// is a workable page rather than 137 days at once.
				if (!oView.getProperty("/from")) {
					var sStart = oVersion.effectiveFrom || "";
					oView.setProperty("/from", sStart);
					oView.setProperty("/to", sStart ? that._addDays(sStart, 13) : "");
				}

				// An actuals container takes actuals; a plan version takes plan
				// values. Default the series to whatever this version accepts.
				if (oVersion.planType === "ACTUAL") {
					oView.setProperty("/series", "ACTUAL");
				}
				that._applyEditability(oDetail);
				return that._loadRows();
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		/** _applyEditability works out whether this user may type into the grid
		 * at all, and says why not when they may not. */
		_applyEditability: function (oDetail) {
			var oView = this.getView().getModel("view");
			var oVersion = oDetail.version;
			var sSeries = oView.getProperty("/series");
			var bReadOnly = false;
			var sReason = "";

			if (sSeries === "ACTUAL") {
				if (oVersion.planType !== "ACTUAL" && oVersion.planType !== "LATEST_ESTIMATE") {
					bReadOnly = true;
					sReason = this.getText("boardActualsElsewhere");
				} else if (!this.can("actual:cane") && !this.can("actual:production") &&
					!this.can("actual:stock") && !this.can("actual:shipment")) {
					bReadOnly = true;
					sReason = this.getText("boardNoActualPermission");
				}
			} else if (!oDetail.editable) {
				bReadOnly = true;
				sReason = oVersion.status === "RELEASED"
					? this.getText("boardReleased", [oVersion.lockedThrough || ""])
					: this.getText("boardNotEditable", [this.formatter.statusText(oVersion.status)]);
			}

			oView.setProperty("/readOnly", bReadOnly);
			oView.setProperty("/readOnlyReason", sReason);
		},

		_index: function (aItems) {
			var mIndex = {};
			(aItems || []).forEach(function (oItem) {
				mIndex[oItem.id] = oItem;
			});
			return mIndex;
		},

		_addDays: function (sDate, iDays) {
			var oDate = new Date(sDate + "T00:00:00Z");
			oDate.setUTCDate(oDate.getUTCDate() + iDays);
			return oDate.toISOString().slice(0, 10);
		},

		_loadRows: function () {
			var oView = this.getView().getModel("view");
			var sKind = oView.getProperty("/kind");
			var that = this;

			this.setBusy(true);
			return this.getService().listRows(this._sVersionId, sKind, {
				from: oView.getProperty("/from"),
				to: oView.getProperty("/to"),
				series: oView.getProperty("/series")
			}).then(function (oPage) {
				var aRows = (oPage.value || []).map(function (oRow) {
					return that._decorate(sKind, oRow);
				});
				that.getView().getModel("rows").setProperty("/" + sKind, aRows);
				oView.setProperty("/rowCount", aRows.length);
				oView.setProperty("/dirty", false);
				that._dirtyKeys = {};
				that._aOriginal = JSON.parse(JSON.stringify(aRows));
				that.setBusy(false);
			});
		},

		/** _decorate adds the names and the calculated columns the grid shows
		 * but the API does not store. */
		_decorate: function (sKind, oRow) {
			var oDecorated = Object.assign({}, oRow);
			if (oRow.productId) {
				oDecorated.productName = (this._mProducts[oRow.productId] || {}).name || oRow.productId;
			}
			if (oRow.warehouseId) {
				oDecorated.warehouseName = (this._mWarehouses[oRow.warehouseId] || {}).code || oRow.warehouseId;
			}
			if (oRow.channelId) {
				oDecorated.channelName = (this._mChannels[oRow.channelId] || {}).name || oRow.channelId;
			}
			if (sKind === "cane") {
				var fAvailable = parseFloat(oRow.availableHours) || 0;
				var fStoppage = parseFloat(oRow.stoppageHours) || 0;
				oDecorated.utilisation = fAvailable > 0
					? ((fAvailable - fStoppage) / fAvailable) * 100
					: 0;
			}
			return oDecorated;
		},

		onKindChange: function () {
			var that = this;
			this._guardUnsaved().then(function (bProceed) {
				if (bProceed) {
					that._loadRows().catch(function (oProblem) { that.showError(oProblem); });
				}
			});
		},

		onRangeChange: function () {
			var that = this;
			this._guardUnsaved().then(function (bProceed) {
				if (!bProceed) {
					return;
				}
				// The series drives what may be edited, so re-evaluate it.
				that.getService().getVersion(that._sVersionId).then(function (oDetail) {
					that._applyEditability(oDetail);
					return that._loadRows();
				}).catch(function (oProblem) {
					that.showError(oProblem);
				});
			});
		},

		/** _guardUnsaved asks before throwing away edits. */
		_guardUnsaved: function () {
			if (!this.getView().getModel("view").getProperty("/dirty")) {
				return Promise.resolve(true);
			}
			return this.confirm(this.getText("discardConfirm"), this.getText("unsavedChanges"));
		},

		onCellChange: function (oEvent) {
			var oContext = oEvent.getSource().getBindingContext("rows");
			if (!oContext) {
				return;
			}
			var oInput = oEvent.getSource();
			var sValue = oInput.getValue();

			if (sValue !== "" && isNaN(parseFloat(sValue))) {
				oInput.setValueState("Error");
				oInput.setValueStateText(this.getText("numericRequired"));
				return;
			}
			if (parseFloat(sValue) < 0) {
				oInput.setValueState("Error");
				oInput.setValueStateText(this.getText("negativeNotAllowed"));
				return;
			}
			oInput.setValueState("None");

			// Track which rows changed so only those are sent.
			var sPath = oContext.getPath();
			this._dirtyKeys[sPath] = true;
			this.getView().getModel("view").setProperty("/dirty", true);
		},

		onDiscard: function () {
			var that = this;
			this.confirm(this.getText("discardConfirm"), this.getText("unsavedChanges"))
				.then(function (bConfirmed) {
					if (bConfirmed) {
						that._loadRows().catch(function (oProblem) { that.showError(oProblem); });
					}
				});
		},

		/**
		 * onSave sends only the changed rows. The request is all-or-nothing by
		 * default, so a planner either saves a consistent week or fixes the one
		 * cell the server objected to.
		 */
		onSave: function () {
			var oView = this.getView().getModel("view");
			var sKind = oView.getProperty("/kind");
			var aAll = this.getView().getModel("rows").getProperty("/" + sKind) || [];
			var that = this;

			var aChanged = Object.keys(this._dirtyKeys).map(function (sPath) {
				var iIndex = parseInt(sPath.split("/").pop(), 10);
				return aAll[iIndex];
			}).filter(Boolean);

			if (!aChanged.length) {
				this.showToast(this.getText("nothingToSave"));
				return;
			}

			var aPayload = aChanged.map(function (oRow) {
				return that._toPayload(sKind, oRow, oView.getProperty("/series"));
			});

			this.setBusy(true);
			this.getService().saveRows(this._sVersionId, sKind, aPayload, false)
				.then(function (oResult) {
					that.showToast(that.getText("rowsSaved", [oResult.accepted]));
					return that._loadRows();
				})
				.catch(function (oProblem) {
					that.setBusy(false);
					that.showError(oProblem);
					that._highlightIssues(oProblem, aChanged, sKind, aAll);
				});
		},

		/** _highlightIssues marks the offending rows so the planner can see
		 * which line the server rejected rather than hunting for it. */
		_highlightIssues: function (oProblem, aChanged, sKind, aAll) {
			if (!oProblem || !oProblem.errors) {
				return;
			}
			var oTable = this.byId(sKind === "cane" ? "caneTable"
				: sKind === "production" ? "productionTable"
					: sKind === "storage" ? "storageTable" : "shipmentTable");
			if (!oTable) {
				return;
			}
			var aItems = oTable.getItems();
			oProblem.errors.forEach(function (oError) {
				if (oError.row === null || oError.row === undefined) {
					return;
				}
				var oRow = aChanged[oError.row];
				var iIndex = aAll.indexOf(oRow);
				if (iIndex >= 0 && aItems[iIndex]) {
					aItems[iIndex].addStyleClass("sugarRowError");
				}
			});
		},

		/** _toPayload strips the display-only fields the API does not accept. */
		_toPayload: function (sKind, oRow, sSeries) {
			var oPayload = { businessDate: oRow.businessDate, series: sSeries };
			var fNum = function (v) {
				var f = parseFloat(v);
				return isNaN(f) ? 0 : f;
			};

			switch (sKind) {
				case "cane":
					Object.assign(oPayload, {
						shiftId: oRow.shiftId || undefined,
						caneAvailable: fNum(oRow.caneAvailable),
						caneDelivered: fNum(oRow.caneDelivered),
						caneAccepted: fNum(oRow.caneAccepted),
						caneRejected: fNum(oRow.caneRejected),
						caneDiverted: fNum(oRow.caneDiverted),
						caneCrushed: fNum(oRow.caneCrushed),
						crushRateTph: fNum(oRow.crushRateTph),
						availableHours: fNum(oRow.availableHours) || 24,
						stoppageHours: fNum(oRow.stoppageHours),
						reasonCode: oRow.reasonCode || undefined,
						note: oRow.note || ""
					});
					break;
				case "production":
					Object.assign(oPayload, {
						productId: oRow.productId,
						packagingId: oRow.packagingId || undefined,
						lineId: oRow.lineId || undefined,
						quantity: fNum(oRow.quantity),
						remeltInput: fNum(oRow.remeltInput),
						processLoss: fNum(oRow.processLoss),
						rework: fNum(oRow.rework),
						rejected: fNum(oRow.rejected),
						holdQty: fNum(oRow.holdQty)
					});
					break;
				case "storage":
					Object.assign(oPayload, {
						warehouseId: oRow.warehouseId,
						productId: oRow.productId,
						productionReceipt: fNum(oRow.productionReceipt),
						transferIn: fNum(oRow.transferIn),
						transferOut: fNum(oRow.transferOut),
						repackIn: fNum(oRow.repackIn),
						repackOut: fNum(oRow.repackOut),
						remeltIssue: fNum(oRow.remeltIssue),
						shipmentQty: fNum(oRow.shipmentQty),
						adjustment: fNum(oRow.adjustment),
						processLoss: fNum(oRow.processLoss),
						holdQty: fNum(oRow.holdQty)
					});
					break;
				case "shipments":
					Object.assign(oPayload, {
						warehouseId: oRow.warehouseId || undefined,
						productId: oRow.productId,
						channelId: oRow.channelId,
						quantity: fNum(oRow.quantity),
						note: oRow.note || ""
					});
					break;
				default:
					break;
			}
			return oPayload;
		},

		onNavBack: function () {
			var that = this;
			this._guardUnsaved().then(function (bProceed) {
				if (bProceed) {
					that.navTo("planDetail", { versionId: that._sVersionId });
				}
			});
		}
	});
});
