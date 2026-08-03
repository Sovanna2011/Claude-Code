sap.ui.define([
    "sap/ui/core/mvc/Controller",
    "sap/ui/core/UIComponent",
    "sap/m/MessageToast",
    "sap/m/MessageBox",
    "hr/module/model/HRService",
    "hr/module/model/formatter"
], function (Controller, UIComponent, MessageToast, MessageBox, HRService, formatter) {
    "use strict";

    return Controller.extend("hr.module.controller.BaseController", {

        formatter: formatter,

        /**
         * Convenience accessor to the shared HR API service, configured with
         * the API base URL from the component config model.
         * @returns {object} the HRService module
         */
        getService: function () {
            var oConfig = this.getOwnerComponent().getModel("config");
            HRService.setBase(oConfig.getProperty("/apiBase"));
            return HRService;
        },

        /** @returns {sap.ui.core.routing.Router} the app router */
        getRouter: function () {
            return UIComponent.getRouterFor(this);
        },

        /** @returns {boolean} whether a user is currently authenticated */
        isAuthenticated: function () {
            return !!this.getOwnerComponent().getModel("auth").getProperty("/authenticated");
        },

        /**
         * Route guard: if not authenticated, redirect to login and return false.
         * @returns {boolean} true if the caller may proceed
         */
        requireAuth: function () {
            if (this.isAuthenticated()) { return true; }
            this.getRouter().navTo("login", {}, true);
            return false;
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
         * Shows an error dialog.
         * @param {string|Error} vError error or message
         */
        showError: function (vError) {
            var sMsg = vError instanceof Error ? vError.message : String(vError);
            MessageBox.error(sMsg);
        },

        /** Signs the current user out and returns to the login screen. */
        onLogout: function () {
            this.getOwnerComponent().logout();
        },

        /** Navigates back to the employee list, or browser history. */
        onNavBack: function () {
            var oHistory = sap.ui.core.routing.History.getInstance();
            var sPrev = oHistory.getPreviousHash();
            if (sPrev !== undefined) {
                window.history.go(-1);
            } else {
                this.getRouter().navTo("employeeList", {}, true);
            }
        }
    });
});
