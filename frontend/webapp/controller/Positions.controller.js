sap.ui.define([
    "hr/module/controller/BaseController",
    "sap/ui/model/json/JSONModel"
], function (BaseController, JSONModel) {
    "use strict";

    return BaseController.extend("hr.module.controller.Positions", {

        onInit: function () {
            this.getView().setModel(new JSONModel({ positions: [] }));
            this.getView().setModel(new JSONModel({ orgUnits: [], orgFilter: null }), "view");
            this.getRouter().getRoute("positions")
                .attachPatternMatched(this._onRouteMatched, this);
        },

        _onRouteMatched: function () {
            if (!this.requireAuth()) { return; }
            var that = this;
            this.getService().getOrgUnits().then(function (aUnits) {
                that.getView().getModel("view").setProperty("/orgUnits", aUnits);
            }).catch(this.showError.bind(this));
            this._loadPositions();
        },

        _loadPositions: function () {
            var oView = this.getView();
            oView.setBusy(true);
            var iOrg = oView.getModel("view").getProperty("/orgFilter");
            this.getService().getPositions(iOrg ? parseInt(iOrg, 10) : null)
                .then(function (aData) {
                    oView.getModel().setProperty("/positions", aData);
                })
                .catch(this.showError.bind(this))
                .finally(function () { oView.setBusy(false); });
        },

        onOrgFilterChange: function () { this._loadPositions(); },

        onRefresh: function () { this._loadPositions(); },

        onNavOrg: function () { this.getRouter().navTo("orgChart"); }
    });
});
