sap.ui.define([
	"sugarplan/controller/BaseController"
], function (BaseController) {
	"use strict";

	return BaseController.extend("sugarplan.controller.Launchpad", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({
				cane: {}, rawSugar: {}, alertCount: 0, asOf: ""
			}), "kpi");

			this.getRouter().getRoute("launchpad").attachPatternMatched(this._onDisplay, this);
			this.onContextRefresh(this._onDisplay);
		},

		_onDisplay: function () {
			var sSeasonId = this.getAppModel().getProperty("/selectedSeasonId");
			if (!sSeasonId) {
				return;
			}
			var that = this;
			this.setBusy(true);
			this.getService().dashboard(sSeasonId)
				.then(function (oDashboard) {
					that.getView().getModel("kpi").setData({
						cane: oDashboard.cane || {},
						rawSugar: oDashboard.rawSugar || {},
						asOf: oDashboard.asOf,
						alertCount: (oDashboard.alerts || []).filter(function (oAlert) {
							return oAlert.severity === "ERROR" || oAlert.severity === "WARNING";
						}).length
					});
					that.setBusy(false);
				})
				.catch(function (oProblem) {
					// A season with no plan yet is a normal state on a fresh
					// installation, not something to interrupt the user with.
					if (oProblem && oProblem.status === 404) {
						that.setBusy(false);
						return;
					}
					that.showError(oProblem);
				});
		},

		onOpenOverview: function () {
			this.navTo("overview");
		},

		onNavigate: function (oEvent) {
			this.navTo(oEvent.getSource().data("route"));
		}
	});
});
