sap.ui.define([
    "sap/ui/core/mvc/Controller",
    "sap/ui/core/UIComponent",
    "sap/ui/core/routing/History",
    "sap/m/MessageToast",
    "sap/m/MessageBox",
    "po/approval/model/POService",
    "po/approval/model/formatter"
], function (Controller, UIComponent, History, MessageToast, MessageBox, POService, formatter) {
    "use strict";

    return Controller.extend("po.approval.controller.BaseController", {

        formatter: formatter,

        /**
         * Convenience accessor to the shared API service, configured with the
         * current base URL and bearer token.
         * @returns {object} the POService module
         */
        getService: function () {
            var oComp = this.getOwnerComponent();
            POService.setBase(oComp.getModel("config").getProperty("/apiBase"));
            POService.setToken(oComp.getModel("auth").getProperty("/token"));
            return POService;
        },

        /** @returns {sap.ui.core.routing.Router} the app router */
        getRouter: function () {
            return UIComponent.getRouterFor(this);
        },

        /**
         * Gets an i18n text.
         * @param {string} sKey resource key
         * @param {array} [aArgs] optional placeholder args
         * @returns {string} translated text
         */
        i18n: function (sKey, aArgs) {
            return this.getOwnerComponent().getModel("i18n")
                .getResourceBundle().getText(sKey, aArgs);
        },

        /**
         * Shows a short toast message.
         * @param {string} sText message text
         */
        toast: function (sText) {
            MessageToast.show(sText);
        },

        /**
         * Shows an error dialog. A 401 additionally ends the session, sending
         * the user back to the login page.
         * @param {Error} oError the error (its .status may carry the HTTP code)
         */
        showError: function (oError) {
            var sMsg = oError instanceof Error ? oError.message : String(oError);
            if (oError && oError.status === 401) {
                this.getOwnerComponent().clearSession();
            }
            MessageBox.error(sMsg);
        },

        /** Navigates back to the worklist, or browser history. */
        onNavBack: function () {
            var sPrev = History.getInstance().getPreviousHash();
            if (sPrev !== undefined) {
                window.history.go(-1);
            } else {
                this.getRouter().navTo("worklist", {}, true);
            }
        },

        /** Logs the current user out. */
        onLogout: function () {
            this.getOwnerComponent().clearSession();
        }
    });
});
