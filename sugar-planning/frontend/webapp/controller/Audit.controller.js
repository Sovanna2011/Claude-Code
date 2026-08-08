sap.ui.define([
	"sugarplan/controller/BaseController"
], function (BaseController) {
	"use strict";

	/**
	 * Audit reads the append-only trail. Only a user with the audit permission
	 * can open it, and nothing on this page can change what it shows.
	 */
	return BaseController.extend("sugarplan.controller.Audit", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({
				events: [], entity: "", action: "", canRead: false
			}), "view");
			this.getRouter().getRoute("audit").attachPatternMatched(this._onDisplay, this);
			this.onContextRefresh(this._onDisplay);
		},

		_onDisplay: function () {
			var bCanRead = this.can("audit:read");
			this.getView().getModel("view").setProperty("/canRead", bCanRead);
			if (!bCanRead) {
				this.getView().getModel("view").setProperty("/events", []);
				return;
			}
			this._load();
		},

		_load: function () {
			var oModel = this.getView().getModel("view");
			var that = this;
			this.setBusy(true);
			this.getService().listAudit({
				entity: oModel.getProperty("/entity"),
				action: oModel.getProperty("/action")
			}).then(function (oPage) {
				oModel.setProperty("/events", oPage.value || []);
				that.setBusy(false);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		onFilterChange: function () {
			this._load();
		},

		onNavBack: function () {
			this.navTo("launchpad");
		}
	});
});
