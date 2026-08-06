sap.ui.define([
	"sugarplan/controller/BaseController",
	"sap/ui/core/Fragment"
], function (BaseController, Fragment) {
	"use strict";

	/**
	 * Maintenance is the outage calendar.
	 *
	 * Approving a window is not a formality: the plan generator treats an
	 * approved, factory-wide window as non-working days, and the season is
	 * extended rather than shortened, so the campaign ends later. The page says
	 * so before anybody presses the button.
	 */
	return BaseController.extend("sugarplan.controller.Maintenance", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({
				windows: [],
				lines: [],
				canWrite: false,
				canApprove: false,
				window: this._emptyWindow()
			}), "view");
			this.getRouter().getRoute("maintenance").attachPatternMatched(this._onDisplay, this);
			this.onContextRefresh(this._onDisplay);
		},

		_emptyWindow: function () {
			var sToday = new Date().toISOString().slice(0, 10);
			return { id: "", lineId: "", startDate: sToday, endDate: sToday, description: "", rowVersion: 0 };
		},

		_onDisplay: function () {
			if (!this.getAppModel().getProperty("/signedIn")) {
				return;
			}
			var oModel = this.getView().getModel("view");
			var oService = this.getService();
			var that = this;

			oModel.setProperty("/canWrite", this.can("downtime:write"));
			oModel.setProperty("/canApprove", this.can("plan:approve"));
			this.setBusy(true);

			Promise.all([
				oService.listMaintenance({ factoryId: this._factoryId() }),
				oService.listMaster("production-lines", { active: "true" })
			]).then(function (aResults) {
				oModel.setProperty("/windows", aResults[0].value || []);
				oModel.setProperty("/lines", aResults[1].value || []);
				that.setBusy(false);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		_factoryId: function () {
			var oProfile = this.getService().getSessionProfile() || {};
			var aFactories = oProfile.factories || [];
			return aFactories.length === 1 ? aFactories[0] : "";
		},

		onNewWindow: function () {
			this.getView().getModel("view").setProperty("/window", this._emptyWindow());
			this._dialog("sugarplan.view.fragment.MaintenanceDialog").then(function (oDialog) {
				oDialog.open();
			});
		},

		onEditWindow: function (oEvent) {
			var oItem = oEvent.getParameter("listItem") || oEvent.getSource();
			var oContext = oItem.getBindingContext("view");
			if (!oContext) {
				return;
			}
			this.getView().getModel("view").setProperty("/window",
				JSON.parse(JSON.stringify(oContext.getObject())));
			this._dialog("sugarplan.view.fragment.MaintenanceDialog").then(function (oDialog) {
				oDialog.open();
			});
		},

		onCancelWindow: function () {
			this._closeDialog("sugarplan.view.fragment.MaintenanceDialog");
		},

		onSaveWindow: function () {
			this._save(this.getView().getModel("view").getProperty("/window"));
		},

		/** onApproveWindow warns what approval costs before it happens: the
		 * campaign gets longer, and somebody has to know that. */
		onApproveWindow: function (oEvent) {
			var oWindow = oEvent.getSource().getBindingContext("view").getObject();
			var iDays = this._dayCount(oWindow.startDate, oWindow.endDate);
			var that = this;

			this.confirm(this.getText("maintenanceApproveConfirm", [iDays]),
				this.getText("maintenanceApproveTitle")).then(function (bOk) {
				if (bOk) {
					var oApproved = JSON.parse(JSON.stringify(oWindow));
					oApproved.status = "APPROVED";
					that._save(oApproved);
				}
			});
		},

		_dayCount: function (sFrom, sTo) {
			var oFrom = new Date(sFrom + "T00:00:00Z");
			var oTo = new Date(sTo + "T00:00:00Z");
			if (isNaN(oFrom.getTime()) || isNaN(oTo.getTime())) {
				return 0;
			}
			return Math.round((oTo - oFrom) / 86400000) + 1;
		},

		_save: function (oWindow) {
			var that = this;
			var oPayload = Object.assign({}, oWindow, { factoryId: this._factoryId() });
			if (!oPayload.id) {
				delete oPayload.id;
			}

			this.setBusy(true);
			this.getService().saveMaintenance(oPayload).then(function () {
				that._closeDialog("sugarplan.view.fragment.MaintenanceDialog");
				that.showToast(that.getText("maintenanceSaved"));
				that._onDisplay();
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		onRefresh: function () {
			this._onDisplay();
		},

		onNavBack: function () {
			this.navTo("launchpad");
		},

		_dialog: function (sName) {
			this._dialogs = this._dialogs || {};
			if (this._dialogs[sName]) {
				return Promise.resolve(this._dialogs[sName]);
			}
			var that = this;
			return Fragment.load({
				id: this.getView().getId(), name: sName, controller: this
			}).then(function (oDialog) {
				that.getView().addDependent(oDialog);
				that._dialogs[sName] = oDialog;
				return oDialog;
			});
		},

		_closeDialog: function (sName) {
			if (this._dialogs && this._dialogs[sName]) {
				this._dialogs[sName].close();
			}
			this.setBusy(false);
		}
	});
});
