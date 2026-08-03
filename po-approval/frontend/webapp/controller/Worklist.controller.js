sap.ui.define([
    "po/approval/controller/BaseController",
    "sap/ui/model/json/JSONModel"
], function (BaseController, JSONModel) {
    "use strict";

    return BaseController.extend("po.approval.controller.Worklist", {

        onInit: function () {
            this.getView().setModel(new JSONModel({ pos: [] }));
            this._sSearch = "";
            this._bOnlyPending = true;
            this.getRouter().getRoute("worklist").attachPatternMatched(this._onRouteMatched, this);
        },

        _onRouteMatched: function () {
            if (this.getOwnerComponent().getModel("auth").getProperty("/token")) {
                this._load();
            }
        },

        _load: function () {
            var oView = this.getView();
            oView.setBusy(true);
            this.getService().getWorklist(this._sSearch, this._bOnlyPending)
                .then(function (aData) {
                    oView.getModel().setProperty("/pos", aData);
                })
                .catch(this.showError.bind(this))
                .finally(function () { oView.setBusy(false); });
        },

        onSearch: function (oEvent) {
            this._sSearch = oEvent.getParameter("query") !== undefined
                ? oEvent.getParameter("query")
                : oEvent.getParameter("newValue");
            this._load();
        },

        onPendingChange: function (oEvent) {
            this._bOnlyPending = oEvent.getParameter("state");
            this._load();
        },

        onRefresh: function () {
            this._load();
        },

        onPoPress: function (oEvent) {
            var oCtx = oEvent.getParameter("listItem").getBindingContext();
            this.getRouter().navTo("poDetail", { ebeln: oCtx.getProperty("ebeln") });
        }
    });
});
