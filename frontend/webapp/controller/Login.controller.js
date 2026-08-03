sap.ui.define([
    "hr/module/controller/BaseController",
    "sap/ui/model/json/JSONModel"
], function (BaseController, JSONModel) {
    "use strict";

    return BaseController.extend("hr.module.controller.Login", {

        onInit: function () {
            this.getView().setModel(new JSONModel({
                username: "", password: "",
                demoUsers: [
                    { name: "admin", pw: "admin123", role: "HR Administrator" },
                    { name: "manager", pw: "manager123", role: "HR Manager" },
                    { name: "linda", pw: "linda123", role: "Employee (Self-Service)" }
                ]
            }));
        },

        onFillDemo: function (oEvent) {
            var oCtx = oEvent.getSource().getBindingContext();
            var oModel = this.getView().getModel();
            oModel.setProperty("/username", oCtx.getProperty("name"));
            oModel.setProperty("/password", oCtx.getProperty("pw"));
        },

        onLogin: function () {
            var oView = this.getView();
            var oData = oView.getModel().getData();
            var oErr = this.byId("loginError");
            oErr.setVisible(false);
            if (!oData.username || !oData.password) {
                oErr.setText(this.i18n("loginRequired")).setVisible(true);
                return;
            }
            var that = this;
            oView.setBusy(true);
            this.getService().login(oData.username, oData.password)
                .then(function (oRes) {
                    that.getOwnerComponent().applyLogin(oRes.token, oRes.user);
                    that.toast(that.i18n("welcome", [oRes.user.displayName || oRes.user.username]));
                    // Route by role: employees land on their own record (self-service).
                    if (oRes.user.roleKey === "EMPLOYEE" && oRes.user.pernr) {
                        that.getRouter().navTo("employeeDetail", { pernr: oRes.user.pernr }, true);
                    } else {
                        that.getRouter().navTo("employeeList", {}, true);
                    }
                })
                .catch(function (e) { oErr.setText(e.message).setVisible(true); })
                .finally(function () { oView.setBusy(false); });
        }
    });
});
