sap.ui.define([
    "hr/module/controller/BaseController",
    "sap/ui/model/json/JSONModel"
], function (BaseController, JSONModel) {
    "use strict";

    return BaseController.extend("hr.module.controller.OrgChart", {

        onInit: function () {
            this.getView().setModel(new JSONModel({ nodes: [] }));
            this.getRouter().getRoute("orgChart")
                .attachPatternMatched(this._loadTree, this);
        },

        _loadTree: function () {
            var oView = this.getView();
            oView.setBusy(true);
            var oSvc = this.getService();
            var that = this;

            // Determine the root org unit(s) from the flat list, then fetch the
            // nested structure for each root and render it as a tree.
            oSvc.getOrgUnits()
                .then(function (aUnits) {
                    var aRoots = aUnits.filter(function (u) {
                        return u.parentOrgId === null || u.parentOrgId === undefined;
                    });
                    if (!aRoots.length && aUnits.length) { aRoots = [aUnits[0]]; }
                    return Promise.all(aRoots.map(function (r) {
                        return oSvc.getOrgStructure(r.orgUnitId);
                    }));
                })
                .then(function (aStructures) {
                    oView.getModel().setProperty("/nodes", aStructures.filter(Boolean));
                })
                .catch(this.showError.bind(this))
                .finally(function () { oView.setBusy(false); });
        },

        onRefresh: function () { this._loadTree(); },

        onNavPositions: function () { this.getRouter().navTo("positions"); }
    });
});
