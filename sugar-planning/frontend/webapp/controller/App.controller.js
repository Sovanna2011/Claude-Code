sap.ui.define([
	"sugarplan/controller/BaseController",
	"sap/m/MessageBox"
], function (BaseController, MessageBox) {
	"use strict";

	/**
	 * App is the shell controller. It owns the session, the season context and
	 * the side navigation; every other page assumes those are in place.
	 */
	return BaseController.extend("sugarplan.controller.App", {

		onInit: function () {
			this.getView().addStyleClass(this.getOwnerComponent().getContentDensityClass());

			// The development accounts, shown on the sign-in page. In an OIDC
			// deployment this list is empty and the sign-in button redirects to
			// the identity provider instead.
			this.getView().setModel(this.getService().newModel({
				users: [
					{ username: "planner", displayName: "Sokha Planner", role: "Production Planner" },
					{ username: "approver", displayName: "Dara Factory Manager", role: "Approver" },
					{ username: "supervisor", displayName: "Vanna Shift Supervisor", role: "Shift Supervisor" },
					{ username: "weighbridge", displayName: "Rithy Weighbridge", role: "Cane Operator" },
					{ username: "warehouse", displayName: "Chanthou Warehouse", role: "Warehouse Operator" },
					{ username: "shipping", displayName: "Sophea Shipment Planner", role: "Shipment Planner" },
					{ username: "executive", displayName: "Bopha Executive", role: "Executive Viewer" },
					{ username: "auditor", displayName: "Sovann Auditor", role: "Auditor" },
					{ username: "admin", displayName: "System Administrator", role: "Administrator" }
				]
			}), "login");

			// A token in session storage means the page was reloaded rather than
			// opened fresh, so the session is resumed silently.
			if (this.getService().hasToken()) {
				this._resumeSession();
			}

			this.getRouter().attachRouteMatched(this._onRouteMatched, this);
		},

		/** _showPage switches the shell between the sign-in page and the
		 * application. sap.m.App shows one page at a time and is driven by
		 * navigation rather than by visibility. */
		_showPage: function (sPageId) {
			var oApp = this.byId("rootApp");
			var oPage = this.byId(sPageId);
			if (oApp && oPage && oApp.getCurrentPage() !== oPage) {
				oApp.to(oPage);
			}
		},

		_onRouteMatched: function (oEvent) {
			this.getAppModel().setProperty("/selectedNav", oEvent.getParameter("name"));
		},

		_resumeSession: function () {
			var that = this;
			this.setBusy(true);
			this.getService().loadSession()
				.then(function (oProfile) {
					return that._afterSignIn(oProfile);
				})
				.catch(function () {
					// An expired token is not an error worth a dialog on load.
					that.getService().logout();
					that.getAppModel().setProperty("/signedIn", false);
					that._showPage("loginPage");
					that.setBusy(false);
				});
		},

		onSignIn: function () {
			var that = this;
			var sUsername = this.byId("userSelect").getSelectedKey();
			if (!sUsername) {
				return;
			}
			this.setBusy(true);
			this.getService().devLogin(sUsername)
				.then(function (oProfile) {
					return that._afterSignIn(oProfile);
				})
				.catch(function (oProblem) {
					that.showError(oProblem);
				});
		},

		/** _afterSignIn loads the season list and lands on the launchpad. */
		_afterSignIn: function (oProfile) {
			var that = this;
			var oApp = this.getAppModel();
			oApp.setProperty("/profile", oProfile);
			oApp.setProperty("/permissions", oProfile.permissions || []);
			oApp.setProperty("/signedIn", true);
			this._showPage("shellPage");

			return this.getService().listSeasons().then(function (oPage) {
				var aSeasons = oPage.value || [];
				oApp.setProperty("/seasons", aSeasons);
				if (aSeasons.length && !oApp.getProperty("/selectedSeasonId")) {
					oApp.setProperty("/selectedSeasonId", aSeasons[0].id);
				}
				that.setBusy(false);
				// The pages mounted before the session existed; tell them the
				// context is ready.
				that.getOwnerComponent().getEventBus().publish("app", "contextChanged");
				if (!aSeasons.length) {
					MessageBox.information(that.getText("noSeasonsForUser"));
				}
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		onSeasonChange: function () {
			// Changing the season invalidates any version selected under it.
			this.getAppModel().setProperty("/selectedVersionId", "");

			// A version-specific page cannot survive a season change; the rest
			// simply reload against the new season.
			var sCurrent = this.getAppModel().getProperty("/selectedNav") || "launchpad";
			if (sCurrent === "planDetail" || sCurrent === "board") {
				this.navTo("plans");
			}
			this.getOwnerComponent().getEventBus().publish("app", "contextChanged");
		},

		onNavItemSelect: function (oEvent) {
			var sKey = oEvent.getParameter("item").getKey();
			this.navTo(sKey);
		},

		onToggleNav: function () {
			var oToolPage = this.byId("toolPage");
			oToolPage.setSideExpanded(!oToolPage.getSideExpanded());
		},

		onOpenProfile: function () {
			var oProfile = this.getAppModel().getProperty("/profile") || {};
			var that = this;
			MessageBox.information(
				this.getText("profileDetail", [
					oProfile.displayName || oProfile.username,
					(oProfile.roles || []).join(", ") || "-",
					(oProfile.permissions || []).length
				]), {
					title: this.getText("profileTitle"),
					actions: [this.getText("signOut"), MessageBox.Action.CLOSE],
					onClose: function (sAction) {
						if (sAction === that.getText("signOut")) {
							that.getService().logout();
							that.getAppModel().setProperty("/signedIn", false);
							that.getAppModel().setProperty("/profile", null);
							that._showPage("loginPage");
						}
					}
				});
		}
	});
});
