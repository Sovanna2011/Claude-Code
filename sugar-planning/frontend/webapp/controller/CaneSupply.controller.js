sap.ui.define([
	"sugarplan/controller/BaseController",
	"sugarplan/model/chart"
], function (BaseController, chart) {
	"use strict";

	/**
	 * Cane supply: where the season's cane comes from.
	 *
	 * The crushing plan says how much cane goes through the mill each day. This
	 * page says which farms it arrives from, when each block is cut, and whether
	 * the lorries exist to move it - and then reconciles the commitments against
	 * the season target, which is the reason the page exists. A cane target with
	 * nothing behind it is a number somebody typed.
	 */
	return BaseController.extend("sugarplan.controller.CaneSupply", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({
				versions: [], versionId: "", actualVersionId: "",
				entries: [], reconciliation: {}, warnings: [],
				scheduleHtml: "", scheduleDays: 0, canWrite: false,
				entry: null, sources: []
			}), "view");
			this.getRouter().getRoute("canesupply").attachPatternMatched(this._onDisplay, this);
			this.onContextRefresh(this._onDisplay);
		},

		_onDisplay: function () {
			var sSeasonId = this.requireSeason();
			if (!sSeasonId) {
				return;
			}
			var oModel = this.getView().getModel("view");
			oModel.setProperty("/canWrite", this.can("plan:write"));

			var that = this;
			this.setBusy(true);
			this.getService().listVersions(sSeasonId).then(function (oPage) {
				var aAll = oPage.value || [];
				// The commitments live on a planning version; the deliveries at
				// the gate are recorded against the actual one. Both are needed
				// to draw one against the other.
				var aPlans = aAll.filter(function (v) { return v.planType !== "ACTUAL"; });
				var oActual = aAll.filter(function (v) { return v.planType === "ACTUAL"; })[0];
				oModel.setProperty("/versions", aPlans);
				oModel.setProperty("/actualVersionId", (oActual || {}).id || "");

				var sSelected = oModel.getProperty("/versionId");
				if (!sSelected || !aPlans.some(function (v) { return v.id === sSelected; })) {
					var oReleased = aPlans.filter(function (v) { return v.status === "RELEASED"; })[0];
					sSelected = (oReleased || aPlans[aPlans.length - 1] || {}).id || "";
					oModel.setProperty("/versionId", sSelected);
				}
				if (!sSelected) {
					that.setBusy(false);
					return null;
				}
				return that._load(sSelected);
			}).catch(function (oProblem) {
				that.setBusy(false);
				that.showError(oProblem);
			});
		},

		_load: function (sVersionId) {
			var that = this;
			var oModel = this.getView().getModel("view");
			return this.getService().supplyPlan(sVersionId).then(function (oPlan) {
				oModel.setProperty("/entries", oPlan.entries || []);
				oModel.setProperty("/reconciliation", oPlan.reconciliation || {});
				oModel.setProperty("/warnings", oPlan.warnings || []);
				return that._drawSchedule(sVersionId);
			}).then(function () {
				that.setBusy(false);
			});
		},

		/**
		 * _drawSchedule draws the delivery schedule with what actually arrived
		 * on top of it.
		 *
		 * The two series are summed per day rather than per source: the queue at
		 * the gate is one queue, and a day where one zone over-delivers while
		 * another fails is still a day the mill was short.
		 */
		_drawSchedule: function (sVersionId) {
			var oModel = this.getView().getModel("view");
			var sActualId = oModel.getProperty("/actualVersionId");
			var that = this;

			var aReads = [this.getService().caneSupply(sVersionId, { series: "PLAN", $top: 20000 })];
			if (sActualId) {
				aReads.push(this.getService().caneSupply(sActualId, { series: "ACTUAL", $top: 20000 }));
			}

			return Promise.all(aReads).then(function (aResponses) {
				var mDays = {};
				var aOrder = [];
				function slot(sDate) {
					if (!mDays[sDate]) {
						mDays[sDate] = { date: sDate, target: 0, actual: 0, hasActual: false };
						aOrder.push(sDate);
					}
					return mDays[sDate];
				}
				(aResponses[0].value || []).forEach(function (oRow) {
					slot(oRow.businessDate).target += parseFloat(oRow.tons) || 0;
				});
				((aResponses[1] || {}).value || []).forEach(function (oRow) {
					var oSlot = slot(oRow.businessDate);
					oSlot.actual += parseFloat(oRow.tons) || 0;
					oSlot.hasActual = true;
				});
				aOrder.sort();
				var aPoints = aOrder.map(function (sDate) { return mDays[sDate]; });

				oModel.setProperty("/scheduleDays", aPoints.length);
				oModel.setProperty("/scheduleHtml", chart.dailyTrend(aPoints, []));
				return that;
			});
		},

		onVersionChange: function () {
			var that = this;
			this.setBusy(true);
			this._load(this.getView().getModel("view").getProperty("/versionId"))
				.catch(function (oProblem) {
					that.setBusy(false);
					that.showError(oProblem);
				});
		},

		/**
		 * onGenerate rebuilds the delivery schedule from the commitments.
		 *
		 * It overwrites the planned rows, so it asks first: a planner who has
		 * hand-adjusted a week of deliveries would lose that work.
		 */
		onGenerate: function () {
			var that = this;
			var sVersionId = this.getView().getModel("view").getProperty("/versionId");
			this.confirm(this.getText("supplyGenerateConfirm"),
				this.getText("supplyGenerate")).then(function (bConfirmed) {
				if (!bConfirmed) {
					return;
				}
				that.setBusy(true);
				that.getService().generateSupplySchedule(sVersionId).then(function (oResult) {
					that.showToast(that.getText("supplyGenerated",
						[oResult.rows, oResult.sources]));
					return that._load(sVersionId);
				}).catch(function (oProblem) {
					that.setBusy(false);
					that.showError(oProblem);
				});
			});
		},

		// --- editing a commitment ------------------------------------------

		onAddEntry: function () {
			this._openEntry({
				id: "", sourceId: "", harvestFrom: "", harvestTo: "",
				committedTons: "0", note: ""
			});
		},

		onEditEntry: function (oEvent) {
			var oLine = oEvent.getSource().getBindingContext("view").getObject();
			this._openEntry({
				id: oLine.id, sourceId: oLine.sourceId,
				harvestFrom: oLine.harvestFrom, harvestTo: oLine.harvestTo,
				committedTons: String(oLine.committedTons), note: oLine.note || ""
			});
		},

		_openEntry: function (oEntry) {
			var that = this;
			var oModel = this.getView().getModel("view");
			oModel.setProperty("/entry", oEntry);

			// The source list is only needed once the dialog opens, so it is not
			// fetched on every visit to the page.
			var pSources = oModel.getProperty("/sources").length
				? Promise.resolve(null)
				: this.getService().listMaster("cane-sources", {})
					.then(function (oPage) {
						oModel.setProperty("/sources", oPage.value || []);
					});

			pSources.then(function () {
				that._dialog("sugarplan.view.fragment.SupplyEntryDialog").then(function (oDialog) { oDialog.open(); });
			}).catch(function (oProblem) { that.showError(oProblem); });
		},

		onCancelEntry: function () {
			this._closeDialog("sugarplan.view.fragment.SupplyEntryDialog");
		},

		onSaveEntry: function () {
			var oModel = this.getView().getModel("view");
			var oEntry = oModel.getProperty("/entry");
			var sVersionId = oModel.getProperty("/versionId");
			var that = this;

			this.setBusy(true);
			this.getService().saveSupplyEntry(sVersionId, {
				id: oEntry.id || undefined,
				versionId: sVersionId,
				sourceId: oEntry.sourceId,
				harvestFrom: oEntry.harvestFrom,
				harvestTo: oEntry.harvestTo,
				committedTons: String(oEntry.committedTons),
				note: oEntry.note
			}).then(function () {
				that._closeDialog("sugarplan.view.fragment.SupplyEntryDialog");
				that.showToast(that.getText("supplySaved"));
				return that._load(sVersionId);
			}).catch(function (oProblem) {
				that.setBusy(false);
				that.showError(oProblem);
			});
		},

		onDeleteEntry: function (oEvent) {
			var oLine = oEvent.getSource().getBindingContext("view").getObject();
			var sVersionId = this.getView().getModel("view").getProperty("/versionId");
			var that = this;
			this.confirm(this.getText("supplyDeleteConfirm", [oLine.sourceCode]),
				this.getText("supplyDelete")).then(function (bConfirmed) {
				if (!bConfirmed) {
					return;
				}
				that.setBusy(true);
				that.getService().deleteSupplyEntry(sVersionId, oLine.id).then(function () {
					that.showToast(that.getText("supplyDeleted"));
					return that._load(sVersionId);
				}).catch(function (oProblem) {
					that.setBusy(false);
					that.showError(oProblem);
				});
			});
		},

		/**
		 * haulageState colours the daily rate a source has to sustain against
		 * the tonnage its lorries can carry.
		 *
		 * This is the number that decides whether a commitment is keepable, so
		 * it is the one that changes colour rather than the commitment itself.
		 */
		haulageState: function (vRequired, vCapacity) {
			var fRequired = parseFloat(vRequired) || 0;
			var fCapacity = parseFloat(vCapacity) || 0;
			if (fCapacity <= 0) {
				return "None";
			}
			if (fRequired > fCapacity) {
				return "Error";
			}
			// Within a tenth of the ceiling there is no room for a breakdown.
			if (fRequired > fCapacity * 0.9) {
				return "Warning";
			}
			return "Success";
		},

		/**
		 * yieldState colours a commitment against what the land should grow.
		 */
		yieldState: function (vCommitted, vExpected) {
			var fCommitted = parseFloat(vCommitted) || 0;
			var fExpected = parseFloat(vExpected) || 0;
			if (fExpected <= 0) {
				return "None";
			}
			return fCommitted > fExpected ? "Error" : "Success";
		},

		coverageState: function (vPct) {
			var fPct = parseFloat(vPct);
			if (isNaN(fPct)) {
				return "None";
			}
			// Short of the target stops the mill; over it only leaves cane
			// standing, so the two are not the same colour.
			if (fPct < 98) {
				return "Error";
			}
			if (fPct > 105) {
				return "Warning";
			}
			return "Success";
		},

		onNavBack: function () {
			this.navTo("launchpad");
		}
	});
});
