sap.ui.define([
    "po/approval/controller/BaseController",
    "sap/ui/model/json/JSONModel"
], function (BaseController, JSONModel) {
    "use strict";

    return BaseController.extend("po.approval.controller.Login", {

        onInit: function () {
            this.getView().setModel(new JSONModel({ username: "", password: "", busy: false }), "login");
            this.getRouter().getRoute("login").attachPatternMatched(this._onRouteMatched, this);
        },

        _onRouteMatched: function () {
            // Reset the form each time the login page is shown.
            this.getView().getModel("login").setData({ username: "", password: "", busy: false });
        },

        onLogin: function () {
            var oModel = this.getView().getModel("login");
            var oData = oModel.getData();
            if (!oData.username || !oData.password) {
                this.showError(new Error(this.i18n("loginValidation")));
                return;
            }
            var that = this;
            oModel.setProperty("/busy", true);
            this.getService().login(oData.username, oData.password)
                .then(function (oLogin) {
                    that.getOwnerComponent().setSession(oLogin);
                    that.toast(that.i18n("loginWelcome", [oLogin.user.displayName]));
                    that.getRouter().navTo("worklist", {}, true);
                })
                .catch(function (oError) {
                    that.showError(oError);
                })
                .finally(function () {
                    oModel.setProperty("/busy", false);
                });
        }
    });
});
