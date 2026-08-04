sap.ui.define([
    "sap/ui/core/UIComponent",
    "sap/ui/Device",
    "sap/ui/model/json/JSONModel",
    "po/approval/model/POService"
], function (UIComponent, Device, JSONModel, POService) {
    "use strict";

    var STORAGE_KEY = "poApprovalAuth";

    return UIComponent.extend("po.approval.Component", {

        metadata: {
            manifest: "json"
        },

        /**
         * Sets up device / config / auth models, restores any persisted
         * session, starts the router and installs the authentication guard.
         */
        init: function () {
            UIComponent.prototype.init.apply(this, arguments);

            var oDeviceModel = new JSONModel(Device);
            oDeviceModel.setDefaultBindingMode("OneWay");
            this.setModel(oDeviceModel, "device");

            this.setModel(new JSONModel({ apiBase: "/api" }), "config");
            POService.setBase("/api");

            // Restore a previously issued token (survives page reloads).
            var oAuth = { token: null, user: null };
            try {
                var sStored = window.sessionStorage.getItem(STORAGE_KEY);
                if (sStored) { oAuth = JSON.parse(sStored); }
            } catch (e) { /* ignore malformed storage */ }
            this.setModel(new JSONModel(oAuth), "auth");
            POService.setToken(oAuth.token);

            this.getRouter().initialize();
            this.getRouter().attachRouteMatched(this._guard, this);
        },

        /**
         * Route guard - forces unauthenticated users to the login page and
         * keeps authenticated users away from it.
         * @param {sap.ui.base.Event} oEvent routeMatched event
         */
        _guard: function (oEvent) {
            var sName = oEvent.getParameter("name");
            var bAuthed = !!this.getModel("auth").getProperty("/token");
            if (!bAuthed && sName !== "login") {
                this.getRouter().navTo("login", {}, true);
            } else if (bAuthed && sName === "login") {
                this.getRouter().navTo("worklist", {}, true);
            }
        },

        /**
         * Persists the issued token + user and configures the API service.
         * @param {object} oLogin the login response {token, user}
         */
        setSession: function (oLogin) {
            var oAuth = { token: oLogin.token, user: oLogin.user };
            this.getModel("auth").setData(oAuth);
            POService.setToken(oLogin.token);
            try {
                window.sessionStorage.setItem(STORAGE_KEY, JSON.stringify(oAuth));
            } catch (e) { /* storage may be unavailable */ }
        },

        /** Clears the session and returns to the login page. */
        clearSession: function () {
            this.getModel("auth").setData({ token: null, user: null });
            POService.setToken(null);
            try {
                window.sessionStorage.removeItem(STORAGE_KEY);
            } catch (e) { /* ignore */ }
            this.getRouter().navTo("login", {}, true);
        },

        getContentDensityClass: function () {
            if (this._sContentDensityClass === undefined) {
                this._sContentDensityClass = !Device.support.touch
                    ? "sapUiSizeCompact" : "sapUiSizeCozy";
            }
            return this._sContentDensityClass;
        }
    });
});
