sap.ui.define([
    "po/approval/controller/BaseController"
], function (BaseController) {
    "use strict";

    return BaseController.extend("po.approval.controller.App", {

        onInit: function () {
            this.getView().addStyleClass(
                this.getOwnerComponent().getContentDensityClass());
        }
    });
});
