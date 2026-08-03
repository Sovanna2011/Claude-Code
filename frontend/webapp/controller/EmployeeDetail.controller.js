sap.ui.define([
    "hr/module/controller/BaseController",
    "sap/ui/model/json/JSONModel",
    "sap/ui/core/Fragment"
], function (BaseController, JSONModel, Fragment) {
    "use strict";

    return BaseController.extend("hr.module.controller.EmployeeDetail", {

        onInit: function () {
            this.getView().setModel(new JSONModel({}));
            this.getView().setModel(new JSONModel({ balances: [] }), "leave");
            this.getRouter().getRoute("employeeDetail")
                .attachPatternMatched(this._onRouteMatched, this);
        },

        _onRouteMatched: function (oEvent) {
            this._pernr = oEvent.getParameter("arguments").pernr;
            this._loadEmployee();
        },

        _loadEmployee: function () {
            var oView = this.getView();
            oView.setBusy(true);
            var that = this;
            this.getService().getEmployee(this._pernr)
                .then(function (oData) {
                    oView.getModel().setData(oData);
                    return that.getService().getLeaveBalances(that._pernr);
                })
                .then(function (aBalances) {
                    oView.getModel("leave").setProperty("/balances", aBalances);
                })
                .catch(this.showError.bind(this))
                .finally(function () { oView.setBusy(false); });
        },

        onTabSelect: function () { /* content is bound; nothing to load lazily */ },

        // ---- Edit Personal Data (IT0002) -----------------------------------
        onOpenEditPersonal: function () {
            var oData = this.getView().getModel().getData();
            var oPd = oData.personalData || {};
            this._openDialog("_pEditDialog", "EditPersonalDialog", function (oView) {
                oView.setModel(new JSONModel({
                    begda: new Date().toISOString().slice(0, 10),
                    formOfAddress: oPd.formOfAddress,
                    firstName: oPd.firstName, lastName: oPd.lastName,
                    middleName: oPd.middleName, birthDate: oPd.birthDate,
                    gender: oPd.genderKey, nationality: oPd.nationality,
                    maritalStatus: oPd.maritalStatusKey,
                    genders: [], maritalStatuses: []
                }), "edit");
                var oSvc = this.getService();
                Promise.all([oSvc.getDomain("GESCH"), oSvc.getDomain("FAMST")])
                    .then(function (aRes) {
                        var oM = oView.getModel("edit");
                        oM.setProperty("/genders", aRes[0]);
                        oM.setProperty("/maritalStatuses", aRes[1]);
                    });
            });
        },

        onEditConfirm: function () {
            var oData = this.getView().getModel("edit").getData();
            var that = this;
            this.getService().updatePersonalData(this._pernr, {
                begda: oData.begda,
                formOfAddress: oData.formOfAddress,
                firstName: oData.firstName,
                lastName: oData.lastName,
                middleName: oData.middleName,
                birthDate: oData.birthDate || null,
                gender: oData.gender,
                nationality: oData.nationality,
                maritalStatus: oData.maritalStatus,
                changedBy: "WEBUI"
            }).then(function () {
                that.toast(that.i18n("saveSuccess"));
                that.byId("editDialog").close();
                that._loadEmployee();
            }).catch(that.showError.bind(that));
        },

        onEditCancel: function () { this.byId("editDialog").close(); },

        // ---- Reassign (IT0001) ---------------------------------------------
        onOpenReassign: function () {
            this._openDialog("_pReassignDialog", "ReassignDialog", function (oView) {
                oView.setModel(new JSONModel({
                    begda: new Date().toISOString().slice(0, 10),
                    orgUnit: null, position: null, costCenter: null,
                    orgUnits: [], positions: []
                }), "reassign");
                var oSvc = this.getService();
                Promise.all([oSvc.getOrgUnits(), oSvc.getPositions()])
                    .then(function (aRes) {
                        var oM = oView.getModel("reassign");
                        oM.setProperty("/orgUnits", aRes[0]);
                        oM.setProperty("/positions", aRes[1]);
                    });
            });
        },

        onReassignConfirm: function () {
            var oData = this.getView().getModel("reassign").getData();
            var that = this;
            this.getService().reassign(this._pernr, {
                begda: oData.begda,
                orgUnit: oData.orgUnit ? parseInt(oData.orgUnit, 10) : null,
                position: oData.position ? parseInt(oData.position, 10) : null,
                costCenter: oData.costCenter || null,
                changedBy: "WEBUI"
            }).then(function () {
                that.toast(that.i18n("saveSuccess"));
                that.byId("reassignDialog").close();
                that._loadEmployee();
            }).catch(that.showError.bind(that));
        },

        onReassignCancel: function () { this.byId("reassignDialog").close(); },

        // ---- Record Absence (IT2001) ---------------------------------------
        onOpenAbsence: function () {
            this._openDialog("_pAbsenceDialog", "RecordAbsenceDialog", function (oView) {
                oView.setModel(new JSONModel({
                    absenceType: "0100", begda: null, endda: null, days: null,
                    absenceTypes: []
                }), "absence");
                this.getService().getValueHelp("absence-types").then(function (aRes) {
                    oView.getModel("absence").setProperty("/absenceTypes", aRes);
                });
            });
        },

        onAbsenceConfirm: function () {
            var oData = this.getView().getModel("absence").getData();
            if (!oData.begda || !oData.endda) {
                this.showError(this.i18n("absenceValidation"));
                return;
            }
            var that = this;
            this.getService().recordAbsence(this._pernr, {
                absenceType: oData.absenceType,
                begda: oData.begda,
                endda: oData.endda,
                days: oData.days ? parseFloat(oData.days) : null,
                changedBy: "WEBUI"
            }).then(function () {
                that.toast(that.i18n("absenceRecorded"));
                that.byId("absenceDialog").close();
                that._loadEmployee();
            }).catch(that.showError.bind(that));
        },

        onAbsenceCancel: function () { this.byId("absenceDialog").close(); },

        // ---- generic dialog loader -----------------------------------------
        _openDialog: function (sCacheKey, sFragmentName, fnBeforeOpen) {
            var oView = this.getView();
            var that = this;
            if (!this[sCacheKey]) {
                this[sCacheKey] = Fragment.load({
                    id: oView.getId(),
                    name: "hr.module.fragment." + sFragmentName,
                    controller: this
                }).then(function (oDialog) {
                    oView.addDependent(oDialog);
                    return oDialog;
                });
            }
            this[sCacheKey].then(function (oDialog) {
                if (fnBeforeOpen) { fnBeforeOpen.call(that, oView); }
                oDialog.open();
            });
        }
    });
});
