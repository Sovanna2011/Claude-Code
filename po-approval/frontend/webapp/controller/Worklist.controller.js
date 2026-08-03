sap.ui.define([
    "po/approval/controller/BaseController",
    "sap/ui/model/json/JSONModel"
], function (BaseController, JSONModel) {
    "use strict";

    return BaseController.extend("po.approval.controller.Worklist", {

        onInit: function () {
            this.getView().setModel(new JSONModel({ pos: [], groups: [] }));
            this.getView().setModel(new JSONModel({ mode: "list", onlyPending: true, assignedToMe: false }), "view");
            this._sSearch = "";
            this.getRouter().getRoute("worklist").attachPatternMatched(this._onRouteMatched, this);
        },

        _onRouteMatched: function () {
            if (this.getOwnerComponent().getModel("auth").getProperty("/token")) {
                this._load();
            }
        },

        _load: function () {
            var oView = this.getView();
            var oV = oView.getModel("view").getData();
            oView.setBusy(true);
            var pLoad = oV.mode === "strategy"
                ? this.getService().getPendingByStrategy(oV.assignedToMe)
                    .then(function (aGroups) {
                        aGroups.forEach(function (g) {
                            g.codesText = (g.codes || []).map(function (c) { return c.code; }).join(" → ");
                        });
                        oView.getModel().setProperty("/groups", aGroups);
                    })
                : this.getService().getWorklist(this._sSearch, oV.onlyPending)
                    .then(function (aData) {
                        oView.getModel().setProperty("/pos", aData);
                    });

            pLoad.catch(this.showError.bind(this))
                .finally(function () { oView.setBusy(false); });
        },

        onModeChange: function (oEvent) {
            this.getView().getModel("view").setProperty("/mode", oEvent.getParameter("item").getKey());
            this._load();
        },

        onSearch: function (oEvent) {
            this._sSearch = oEvent.getParameter("query") !== undefined
                ? oEvent.getParameter("query")
                : oEvent.getParameter("newValue");
            this._load();
        },

        onPendingChange: function (oEvent) {
            this.getView().getModel("view").setProperty("/onlyPending", oEvent.getParameter("selected"));
            this._load();
        },

        onAssignedChange: function (oEvent) {
            this.getView().getModel("view").setProperty("/assignedToMe", oEvent.getParameter("selected"));
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
