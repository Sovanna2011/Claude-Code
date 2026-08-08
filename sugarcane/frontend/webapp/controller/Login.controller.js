sap.ui.define([
	"sap/ui/core/mvc/Controller"
], function (Controller) {
	"use strict";

	return Controller.extend("farm.area.dashboard.controller.Login", {
		onInit: function () {
			// A token from an earlier tab is still good: go straight in rather than asking again.
			if (this.getOwnerComponent().getApi().isSignedIn()) {
				this._enter(this.getOwnerComponent().getApi().currentUser());
			}
		},

		onSignIn: function () {
			var view = this.getView();
			var strip = view.byId("loginError");
			var button = view.byId("signIn");

			strip.setVisible(false);
			button.setBusy(true);

			this.getOwnerComponent().getApi()
				.login(view.byId("userName").getValue().trim(), view.byId("password").getValue())
				.then(function (user) {
					button.setBusy(false);
					this._enter(user);
				}.bind(this))
				.catch(function (error) {
					button.setBusy(false);
					strip.setText(error.message || "Sign-in failed.");
					strip.setVisible(true);
				});
		},

		_enter: function (user) {
			this.getOwnerComponent().getModel("session").setData({ user: user, signedIn: true });
			this.getOwnerComponent().getRouter().navTo("dashboard");
		}
	});
});
