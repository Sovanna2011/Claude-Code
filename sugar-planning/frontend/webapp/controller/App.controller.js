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

			// The development accounts are read from the server rather than
			// listed here, so an account added to the configuration and
			// forgotten in the browser cannot happen. In an OIDC deployment the
			// list comes back empty and the sign-in button redirects to the
			// identity provider instead.
			this.getView().setModel(this.getService().newModel({ users: [] }), "login");
			this._loadDevUsers();

			// A token in session storage means the page was reloaded rather than
			// opened fresh, so the session is resumed silently.
			if (this.getService().hasToken()) {
				this._resumeSession();
			}

			this.getRouter().attachRouteMatched(this._onRouteMatched, this);
		},

		/** _loadDevUsers fills the sign-in list from the API. A failure is not
		 * worth an error dialog on a page nobody has signed in to yet; the list
		 * stays empty and the sign-in button says nothing is available. */
		_loadDevUsers: function () {
			var that = this;
			this.getService().get("/auth/dev-users").then(function (oPage) {
				that.getView().getModel("login").setProperty("/users", (oPage.value || []).map(function (oUser) {
					return {
						username: oUser.username,
						displayName: oUser.displayName || oUser.username,
						role: (oUser.roles || []).map(that._roleLabel).join(", ")
					};
				}));
			}).catch(function () {
				that.getView().getModel("login").setProperty("/users", []);
			});
		},

		/** _roleLabel turns SHIFT_SUPERVISOR into "Shift Supervisor". */
		_roleLabel: function (sRole) {
			return String(sRole).toLowerCase().split("_").map(function (sWord) {
				return sWord.charAt(0).toUpperCase() + sWord.slice(1);
			}).join(" ");
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
				that._refreshInbox();
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

		/**
		 * onOpenInbox shows what has been raised for the roles this person
		 * holds, and marks read whatever they open.
		 *
		 * It is a dialog rather than a page because an inbox is read in the
		 * middle of doing something else - that is what makes it an inbox - and
		 * navigating away from a half-finished screen to look at it would be the
		 * wrong trade.
		 */
		onOpenInbox: function () {
			var that = this;
			this.getService().listNotifications(false).then(function (oPage) {
				that.getAppModel().setProperty("/unread", oPage.unread || 0);
				that.getAppModel().setProperty("/notifications", oPage.value || []);
				that._dialog("sugarplan.view.fragment.InboxDialog").then(function (oDialog) {
					oDialog.open();
				});
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		/** onReadNotification clears one and refreshes the badge. */
		onReadNotification: function (oEvent) {
			var oItem = oEvent.getSource().getBindingContext("app").getObject();
			var that = this;
			this.getService().markNotificationRead(oItem.id).then(function (oResult) {
				that.getAppModel().setProperty("/unread", oResult.unread || 0);
				return that.getService().listNotifications(false);
			}).then(function (oPage) {
				that.getAppModel().setProperty("/notifications", oPage.value || []);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		onCloseInbox: function () {
			this._closeDialog("sugarplan.view.fragment.InboxDialog");
		},

		/** _refreshInbox keeps the badge current without opening anything. */
		_refreshInbox: function () {
			var that = this;
			this.getService().listNotifications(true).then(function (oPage) {
				that.getAppModel().setProperty("/unread", oPage.unread || 0);
			}).catch(function () {
				// A badge that cannot be fetched is not worth interrupting
				// somebody for; the inbox itself will say so when they open it.
			});
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
