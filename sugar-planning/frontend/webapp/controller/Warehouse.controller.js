sap.ui.define([
	"sugarplan/controller/BaseController",
	"sugarplan/model/chart"
], function (BaseController, chart) {
	"use strict";

	/**
	 * Warehouse answers the capacity question: how full is each store, when
	 * does it fill, and what shipment rate would keep it inside its limits.
	 */
	return BaseController.extend("sugarplan.controller.Warehouse", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({
				storage: [], ledger: [], chartHtml: "", selectedId: "", selectedName: ""
			}), "view");
			this.getRouter().getRoute("warehouse").attachPatternMatched(this._onDisplay, this);
			this.onContextRefresh(this._onDisplay);
		},

		_onDisplay: function (oEvent) {
			var sSeasonId = this.requireSeason();
			if (!sSeasonId) {
				return;
			}
			// A drill-down from the dashboard names the store it was describing,
			// so the ledger opens on it rather than making the reader find the
			// same row again.
			var sWanted = (oEvent && oEvent.getParameter("arguments")
				&& oEvent.getParameter("arguments")["?query"]
				&& oEvent.getParameter("arguments")["?query"].warehouseId) || "";
			var that = this;
			this.setBusy(true);

			this.getService().dashboard(sSeasonId).then(function (oDashboard) {
				that._sVersionId = (oDashboard.planVersion || {}).id;
				var aStorage = oDashboard.storage || [];
				that.getView().getModel("view").setProperty("/storage", aStorage);
				that.setBusy(false);

				var oWanted = aStorage.filter(function (oStore) {
					return oStore.warehouseId === sWanted;
				})[0];
				if (oWanted) {
					that._showLedger(oWanted);
				}
			}).catch(function (oProblem) {
				if (oProblem && oProblem.status === 404) {
					that.setBusy(false);
					return;
				}
				that.showError(oProblem);
			});
		},

		/** onSelectWarehouse loads the daily ledger of one store and draws its
		 * balance against the capacity lines. */
		onSelectWarehouse: function (oEvent) {
			var oItem = oEvent.getParameter("listItem");
			if (!oItem) {
				return;
			}
			this._showLedger(oItem.getBindingContext("view").getObject());
		},

		/** _showLedger loads one store's daily ledger, whether the reader picked
		 * it here or arrived from a KPI that named it. */
		_showLedger: function (oStore) {
			var oModel = this.getView().getModel("view");
			var that = this;

			oModel.setProperty("/selectedId", oStore.warehouseId);
			oModel.setProperty("/selectedName", oStore.warehouseCode + " — " + oStore.warehouseName);

			this.setBusy(true);
			this.getService().listRows(this._sVersionId, "storage", {
				warehouseId: oStore.warehouseId,
				series: "PLAN"
			}).then(function (oPage) {
				var aRows = oPage.value || [];

				// One store can hold several products; the capacity is shared,
				// so the curve sums them per day.
				var mByDate = {};
				aRows.forEach(function (oRow) {
					mByDate[oRow.businessDate] = (mByDate[oRow.businessDate] || 0) +
						(parseFloat(oRow.endingBalance) || 0);
				});
				var aDays = Object.keys(mByDate).sort().map(function (sDate) {
					return { date: sDate, endingBalance: mByDate[sDate] };
				});

				oModel.setProperty("/ledger", aRows);
				oModel.setProperty("/chartHtml",
					chart.stockCurve(aDays, oStore.usableCapacityTons, 80));
				that.setBusy(false);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		onRefresh: function () {
			this._onDisplay();
		},

		onNavBack: function () {
			this.navTo("launchpad");
		}
	});
});
