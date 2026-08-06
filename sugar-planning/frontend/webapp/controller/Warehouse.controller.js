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

		_onDisplay: function () {
			var sSeasonId = this.requireSeason();
			if (!sSeasonId) {
				return;
			}
			var that = this;
			this.setBusy(true);

			this.getService().dashboard(sSeasonId).then(function (oDashboard) {
				that._sVersionId = (oDashboard.planVersion || {}).id;
				that.getView().getModel("view").setProperty("/storage", oDashboard.storage || []);
				that.setBusy(false);
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
			var oStore = oItem.getBindingContext("view").getObject();
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
