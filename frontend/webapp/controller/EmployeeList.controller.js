sap.ui.define([
    "hr/module/controller/BaseController",
    "sap/ui/model/json/JSONModel",
    "sap/ui/core/Fragment"
], function (BaseController, JSONModel, Fragment) {
    "use strict";

    return BaseController.extend("hr.module.controller.EmployeeList", {

        onInit: function () {
            this.getView().setModel(new JSONModel({ employees: [] }));
            this._sSearch = "";
            this._sKeyDate = null;
            this.getRouter().getRoute("employeeList")
                .attachPatternMatched(this._onRouteMatched, this);
        },

        _onRouteMatched: function () {
            this._loadEmployees();
        },

        _loadEmployees: function () {
            var oView = this.getView();
            oView.setBusy(true);
            this.getService().getEmployees(this._sSearch, this._sKeyDate)
                .then(function (aData) {
                    oView.getModel().setProperty("/employees", aData);
                })
                .catch(this.showError.bind(this))
                .finally(function () { oView.setBusy(false); });
        },

        onSearch: function (oEvent) {
            this._sSearch = oEvent.getParameter("query") !== undefined
                ? oEvent.getParameter("query")
                : oEvent.getParameter("newValue");
            this._loadEmployees();
        },

        onKeyDateChange: function (oEvent) {
            this._sKeyDate = oEvent.getParameter("value") || null;
            this._loadEmployees();
        },

        onRefresh: function () {
            this._loadEmployees();
        },

        onEmployeePress: function (oEvent) {
            var oCtx = oEvent.getParameter("listItem").getBindingContext();
            this.getRouter().navTo("employeeDetail", { pernr: oCtx.getProperty("pernr") });
        },

        onNavOrg: function () {
            this.getRouter().navTo("orgChart");
        },

        onNavPositions: function () {
            this.getRouter().navTo("positions");
        },

        // ---- Hire dialog ---------------------------------------------------
        onOpenHire: function () {
            var oView = this.getView();
            if (!this._pHireDialog) {
                this._pHireDialog = Fragment.load({
                    id: oView.getId(),
                    name: "hr.module.fragment.HireEmployeeDialog",
                    controller: this
                }).then(function (oDialog) {
                    oView.addDependent(oDialog);
                    return oDialog;
                });
            }
            var that = this;
            this._pHireDialog.then(function (oDialog) {
                // Reset the form model and load value helps.
                oView.setModel(new JSONModel({
                    hireDate: new Date().toISOString().slice(0, 10),
                    firstName: "", lastName: "", gender: "1",
                    companyCode: "1000", personnelArea: "1000",
                    employeeGroup: "1", employeeSubgroup: "DU",
                    email: "",
                    companyCodes: [], personnelAreas: [], employeeGroups: [],
                    employeeSubgroups: [], genders: [], orgUnits: []
                }), "hire");
                that._loadHireValueHelps();
                oDialog.open();
            });
        },

        _loadHireValueHelps: function () {
            var oModel = this.getView().getModel("hire");
            var oSvc = this.getService();
            Promise.all([
                oSvc.getValueHelp("company-codes"),
                oSvc.getValueHelp("personnel-areas"),
                oSvc.getValueHelp("employee-groups"),
                oSvc.getValueHelp("employee-subgroups"),
                oSvc.getDomain("GESCH"),
                oSvc.getOrgUnits()
            ]).then(function (aRes) {
                oModel.setProperty("/companyCodes", aRes[0]);
                oModel.setProperty("/personnelAreas", aRes[1]);
                oModel.setProperty("/employeeGroups", aRes[2]);
                oModel.setProperty("/employeeSubgroups", aRes[3]);
                oModel.setProperty("/genders", aRes[4]);
                oModel.setProperty("/orgUnits", aRes[5]);
            }).catch(this.showError.bind(this));
        },

        onHireConfirm: function () {
            var oData = this.getView().getModel("hire").getData();
            if (!oData.firstName || !oData.lastName || !oData.hireDate) {
                this.showError(this.i18n("hireValidation"));
                return;
            }
            var that = this;
            this.getService().hireEmployee({
                hireDate: oData.hireDate,
                firstName: oData.firstName,
                lastName: oData.lastName,
                gender: oData.gender,
                birthDate: oData.birthDate || null,
                companyCode: oData.companyCode,
                personnelArea: oData.personnelArea,
                employeeGroup: oData.employeeGroup,
                employeeSubgroup: oData.employeeSubgroup,
                orgUnit: oData.orgUnit ? parseInt(oData.orgUnit, 10) : null,
                email: oData.email || null,
                changedBy: "WEBUI"
            }).then(function (oRes) {
                that.toast(oRes.message);
                that.byId("hireDialog").close();
                that._loadEmployees();
                that.getRouter().navTo("employeeDetail", { pernr: oRes.pernr });
            }).catch(that.showError.bind(that));
        },

        onHireCancel: function () {
            this.byId("hireDialog").close();
        }
    });
});
