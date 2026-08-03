sap.ui.define([
    "hr/module/controller/BaseController"
], function (BaseController) {
    "use strict";

    return BaseController.extend("hr.module.controller.App", {

        onInit: function () {
            // Apply the content density class chosen by the component.
            this.getView().addStyleClass(
                this.getOwnerComponent().getContentDensityClass());
        }
    });
});
