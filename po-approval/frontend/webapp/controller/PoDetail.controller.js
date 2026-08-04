sap.ui.define([
    "po/approval/controller/BaseController",
    "sap/ui/model/json/JSONModel",
    "sap/ui/core/Fragment",
    "sap/m/MessageBox"
], function (BaseController, JSONModel, Fragment, MessageBox) {
    "use strict";

    return BaseController.extend("po.approval.controller.PoDetail", {

        onInit: function () {
            this.getView().setModel(new JSONModel({}));
            this.getRouter().getRoute("poDetail").attachPatternMatched(this._onRouteMatched, this);
        },

        _onRouteMatched: function (oEvent) {
            this._sEbeln = oEvent.getParameter("arguments").ebeln;
            if (this.getOwnerComponent().getModel("auth").getProperty("/token")) {
                this._load();
            }
        },

        _load: function () {
            var oView = this.getView();
            oView.setBusy(true);
            this.getService().getPurchaseOrder(this._sEbeln)
                .then(function (oData) {
                    oView.getModel().setData(oData);
                })
                .catch(this.showError.bind(this))
                .finally(function () { oView.setBusy(false); });
        },

        onRelease: function () {
            var that = this;
            MessageBox.confirm(this.i18n("confirmRelease", [this._sEbeln]), {
                onClose: function (sAction) {
                    if (sAction !== MessageBox.Action.OK) { return; }
                    that.getView().setBusy(true);
                    that.getService().release(that._sEbeln, {})
                        .then(function (oData) {
                            that.getView().getModel().setData(oData);
                            that.toast(that.i18n("releaseDone"));
                        })
                        .catch(that.showError.bind(that))
                        .finally(function () { that.getView().setBusy(false); });
                }
            });
        },

        onReject: function () {
            var oView = this.getView();
            if (!this._pRejectDialog) {
                this._pRejectDialog = Fragment.load({
                    id: oView.getId(),
                    name: "po.approval.fragment.RejectDialog",
                    controller: this
                }).then(function (oDialog) {
                    oView.addDependent(oDialog);
                    return oDialog;
                });
            }
            var that = this;
            this._pRejectDialog.then(function (oDialog) {
                oView.setModel(new JSONModel({ note: "" }), "reject");
                oDialog.open();
            });
        },

        onRejectConfirm: function () {
            var oNote = this.getView().getModel("reject").getProperty("/note");
            if (!oNote) {
                this.showError(new Error(this.i18n("rejectReasonRequired")));
                return;
            }
            var that = this;
            this.byId("rejectDialog").setBusy(true);
            this.getService().reject(this._sEbeln, { note: oNote })
                .then(function (oData) {
                    that.getView().getModel().setData(oData);
                    that.toast(that.i18n("rejectDone"));
                    that.byId("rejectDialog").close();
                })
                .catch(that.showError.bind(that))
                .finally(function () { that.byId("rejectDialog").setBusy(false); });
        },

        onRejectCancel: function () {
            this.byId("rejectDialog").close();
        }
    });
});
