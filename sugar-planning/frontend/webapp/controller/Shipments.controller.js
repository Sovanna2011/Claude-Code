sap.ui.define([
	"sugarplan/controller/BaseController"
], function (BaseController) {
	"use strict";

	/**
	 * Shipments shows planned against actual dispatch by channel, and what the
	 * capacity position implies for the shipment rate.
	 */
	return BaseController.extend("sugarplan.controller.Shipments", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({
				channels: [], storage: [], totalPlanned: 0, totalActual: 0
			}), "view");
			this.getRouter().getRoute("shipments").attachPatternMatched(this._onDisplay, this);
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
				var aChannels = oDashboard.shipments || [];
				var fPlanned = 0, fActual = 0;
				aChannels.forEach(function (oChannel) {
					fPlanned += parseFloat(oChannel.plannedTons) || 0;
					fActual += parseFloat(oChannel.actualTons) || 0;
				});
				that.getView().getModel("view").setData({
					channels: aChannels,
					storage: (oDashboard.storage || []).filter(function (oStore) {
						return oStore.storageClass === "FINISHED";
					}),
					totalPlanned: fPlanned,
					totalActual: fActual
				});
				that.setBusy(false);
			}).catch(function (oProblem) {
				if (oProblem && oProblem.status === 404) {
					that.setBusy(false);
					return;
				}
				that.showError(oProblem);
			});
		},

		onNavBack: function () {
			this.navTo("launchpad");
		}
	});
});
