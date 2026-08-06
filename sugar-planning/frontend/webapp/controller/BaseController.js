sap.ui.define([
	"sap/ui/core/mvc/Controller",
	"sap/ui/core/Fragment",
	"sap/m/MessageBox",
	"sap/m/MessageToast",
	"sugarplan/model/formatter"
], function (Controller, Fragment, MessageBox, MessageToast, formatter) {
	"use strict";

	/**
	 * BaseController holds what every page needs: access to the service and the
	 * router, and one consistent way of reporting an error.
	 */
	return Controller.extend("sugarplan.controller.BaseController", {

		formatter: formatter,

		getService: function () {
			return this.getOwnerComponent().getService();
		},

		getRouter: function () {
			return this.getOwnerComponent().getRouter();
		},

		getAppModel: function () {
			return this.getOwnerComponent().getModel("app");
		},

		/**
		 * onContextRefresh runs fn whenever the session or the selected season
		 * changes.
		 *
		 * A page's route can match before sign-in has finished, and the season
		 * can be switched in the shell while a page is already open. Both cases
		 * mean "reload with the current context", so both raise one event and
		 * every page listens for it.
		 */
		onContextRefresh: function (fn) {
			this._fnContextRefresh = fn.bind(this);
			this.getOwnerComponent().getEventBus()
				.subscribe("app", "contextChanged", this._fnContextRefresh, this);
		},

		onExit: function () {
			if (this._fnContextRefresh) {
				this.getOwnerComponent().getEventBus()
					.unsubscribe("app", "contextChanged", this._fnContextRefresh, this);
			}
		},

		getText: function (sKey, aArgs) {
			return this.getOwnerComponent().getModel("i18n").getResourceBundle().getText(sKey, aArgs);
		},

		setBusy: function (bBusy) {
			this.getAppModel().setProperty("/busy", !!bBusy);
		},

		navTo: function (sRoute, oParams) {
			this.getRouter().navTo(sRoute, oParams);
		},

		/**
		 * showError renders a problem document.
		 *
		 * The API returns RFC 9457 problems with field-level errors addressed by
		 * row and field, so a rejected planning grid save can name the exact
		 * cell rather than saying "invalid input".
		 */
		showError: function (oProblem) {
			this.setBusy(false);

			if (!oProblem) {
				MessageBox.error(this.getText("errorUnknown"));
				return;
			}
			if (oProblem.status === 401) {
				this.getService().logout();
				this.getAppModel().setProperty("/signedIn", false);
				MessageBox.warning(this.getText("errorSessionExpired"));
				return;
			}

			var sTitle = oProblem.title || this.getText("errorTitle");
			var sDetail = oProblem.detail || "";

			if (oProblem.errors && oProblem.errors.length) {
				var aLines = oProblem.errors.slice(0, 12).map(function (oError) {
					var sWhere = oError.field || "";
					if (oError.row !== null && oError.row !== undefined) {
						sWhere = "Row " + (oError.row + 1) + (sWhere ? ", " + sWhere : "");
					}
					return (sWhere ? sWhere + ": " : "") + oError.message;
				});
				if (oProblem.errors.length > 12) {
					aLines.push("... and " + (oProblem.errors.length - 12) + " more");
				}
				sDetail = (sDetail ? sDetail + "\n\n" : "") + aLines.join("\n");
			}
			if (oProblem.correlationId) {
				sDetail += "\n\nReference: " + oProblem.correlationId;
			}

			MessageBox.error(sDetail || sTitle, { title: sTitle });
		},

		showToast: function (sMessage) {
			MessageToast.show(sMessage);
		},

		/** confirm asks before an action that is hard to undo. */
		confirm: function (sMessage, sTitle) {
			return new Promise(function (resolve) {
				MessageBox.confirm(sMessage, {
					title: sTitle,
					onClose: function (sAction) {
						resolve(sAction === MessageBox.Action.OK);
					}
				});
			});
		},

		/** requireSeason returns the selected season, sending the user to the
		 * launchpad if none has been chosen yet. */
		requireSeason: function () {
			var sSeasonId = this.getAppModel().getProperty("/selectedSeasonId");
			if (!sSeasonId) {
				this.navTo("launchpad");
				return null;
			}
			return sSeasonId;
		},

		/** can reports whether the signed-in user holds a permission. */
		/**
		 * _dialog loads a fragment once and keeps it.
		 *
		 * A dialog reloaded on every open loses its state and leaks a control
		 * tree each time, so it is cached against the controller and made a
		 * dependent of the view - which is what disposes of it when the page
		 * goes away.
		 */
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
		},

		can: function (sPermission) {
			return this.getService().can(sPermission);
		}
	});
});
