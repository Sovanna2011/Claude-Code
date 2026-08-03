sap.ui.define([
    "sap/ui/core/UIComponent",
    "sap/ui/Device",
    "sap/ui/model/json/JSONModel"
], function (UIComponent, Device, JSONModel) {
    "use strict";

    return UIComponent.extend("hr.module.Component", {

        metadata: {
            manifest: "json"
        },

        /**
         * Called on component initialisation. Sets up the device model,
         * the API base-url config model, and starts the router.
         */
        init: function () {
            UIComponent.prototype.init.apply(this, arguments);

            // Device model for responsive behaviour.
            var oDeviceModel = new JSONModel(Device);
            oDeviceModel.setDefaultBindingMode("OneWay");
            this.setModel(oDeviceModel, "device");

            // Config model - central place for the API base URL. Adjust when
            // the API is not served from the same origin as the UI5 app.
            this.setModel(new JSONModel({
                apiBase: "/api",
                keyDate: null
            }), "config");

            this.getRouter().initialize();
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
