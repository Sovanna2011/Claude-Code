sap.ui.define([
    "sap/ui/core/UIComponent",
    "sap/ui/Device",
    "sap/ui/model/json/JSONModel",
    "hr/module/model/HRService"
], function (UIComponent, Device, JSONModel, HRService) {
    "use strict";

    return UIComponent.extend("hr.module.Component", {

        metadata: {
            manifest: "json"
        },

        /**
         * Called on component initialisation. Sets up the device model,
         * the API base-url config model, authentication, and starts the router.
         */
        init: function () {
            UIComponent.prototype.init.apply(this, arguments);

            // Device model for responsive behaviour.
            var oDeviceModel = new JSONModel(Device);
            oDeviceModel.setDefaultBindingMode("OneWay");
            this.setModel(oDeviceModel, "device");

            // Config model - central place for the API base URL.
            this.setModel(new JSONModel({ apiBase: "/api", keyDate: null }), "config");

            // Auth model - current user + role (drives visibility).
            this.setModel(new JSONModel({
                authenticated: false, username: null, displayName: null,
                roleKey: null, roleName: null, pernr: null,
                isAdmin: false, isManager: false, isEmployee: false
            }), "auth");

            HRService.setBase("/api");
            // On any 401, drop the session and return to the login screen.
            HRService.setUnauthorizedHandler(function () { this.logout(); }.bind(this));

            // Restore a saved session (survives refresh within the tab).
            var sSaved = window.sessionStorage.getItem("hr.session");
            if (sSaved) {
                try {
                    var o = JSON.parse(sSaved);
                    this.applyLogin(o.token, o.user, true);
                } catch (e) { /* ignore corrupt session */ }
            }

            this.getRouter().initialize();

            // Guard: if not authenticated, force the login screen.
            if (!this.getModel("auth").getProperty("/authenticated")) {
                this.getRouter().navTo("login", {}, true);
            }
        },

        /** Applies a successful login: stores the token and populates the auth model. */
        applyLogin: function (sToken, oUser, bSilent) {
            HRService.setToken(sToken);
            this.getModel("auth").setData({
                authenticated: true,
                username: oUser.username, displayName: oUser.displayName || oUser.username,
                roleKey: oUser.roleKey, roleName: oUser.roleName, pernr: oUser.pernr,
                isAdmin: oUser.roleKey === "HR_ADMIN",
                isManager: oUser.roleKey === "HR_MANAGER",
                isEmployee: oUser.roleKey === "EMPLOYEE"
            });
            if (!bSilent) {
                window.sessionStorage.setItem("hr.session", JSON.stringify({ token: sToken, user: oUser }));
            }
        },

        /** Clears the session and returns to the login screen. */
        logout: function () {
            HRService.setToken(null);
            window.sessionStorage.removeItem("hr.session");
            this.getModel("auth").setProperty("/authenticated", false);
            this.getRouter().navTo("login", {}, true);
        },

        /**
         * Returns the content density class matching the current device.
         * @returns {string} density css class
         */
        getContentDensityClass: function () {
            if (this._sContentDensityClass === undefined) {
                if (!Device.support.touch) {
                    this._sContentDensityClass = "sapUiSizeCompact";
                } else {
                    this._sContentDensityClass = "sapUiSizeCozy";
                }
            }
            return this._sContentDensityClass;
        }
    });
});
