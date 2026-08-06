sap.ui.define([
	"sugarplan/controller/BaseController"
], function (BaseController) {
	"use strict";

	/**
	 * Materials shows the packaging requirement that follows from the plan, and
	 * the date by which each order has to be placed to keep the line running.
	 */
	return BaseController.extend("sugarplan.controller.Materials", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({
				requirements: [], versions: [], versionId: "", shortages: 0
			}), "view");
			this.getRouter().getRoute("materials").attachPatternMatched(this._onDisplay, this);
			this.onContextRefresh(this._onDisplay);
		},

		_onDisplay: function () {
			var sSeasonId = this.requireSeason();
			if (!sSeasonId) {
				return;
			}
			var that = this;
			this.setBusy(true);

			this.getService().listVersions(sSeasonId).then(function (oPage) {
				var aVersions = (oPage.value || []).filter(function (oVersion) {
					return oVersion.planType !== "ACTUAL";
				});
				that.getView().getModel("view").setProperty("/versions", aVersions);
				var sSelected = that.getView().getModel("view").getProperty("/versionId");
				if (!sSelected || !aVersions.some(function (v) { return v.id === sSelected; })) {
					var oReleased = aVersions.filter(function (v) { return v.status === "RELEASED"; })[0];
					sSelected = (oReleased || aVersions[aVersions.length - 1] || {}).id || "";
					that.getView().getModel("view").setProperty("/versionId", sSelected);
				}
				if (!sSelected) {
					that.setBusy(false);
					return null;
				}
				return that._load(sSelected);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		_load: function (sVersionId) {
			var that = this;
			return this.getService().materialRequirements(sVersionId).then(function (oPage) {
				var aRequirements = oPage.value || [];
				that.getView().getModel("view").setProperty("/requirements", aRequirements);
				that.getView().getModel("view").setProperty("/shortages",
					aRequirements.filter(function (oRequirement) {
						return oRequirement.severity === "ERROR";
					}).length);
				that.setBusy(false);
			});
		},

		onVersionChange: function () {
			var that = this;
			this.setBusy(true);
			this._load(this.getView().getModel("view").getProperty("/versionId"))
				.catch(function (oProblem) { that.showError(oProblem); });
		},

		onNavBack: function () {
			this.navTo("launchpad");
		}
	});
});
