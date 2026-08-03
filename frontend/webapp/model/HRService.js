sap.ui.define([
    "sap/ui/model/json/JSONModel"
], function (JSONModel) {
    "use strict";

    /**
     * Thin wrapper over the HR Module REST API. All methods return Promises
     * resolving to the parsed JSON payload (or rejecting with an Error whose
     * message is the API error text).
     */
    return {

        _base: "/api",

        /**
         * Sets the API base URL (from the config model).
         * @param {string} sBase base url, e.g. "/api"
         */
        setBase: function (sBase) {
            if (sBase) { this._base = sBase.replace(/\/$/, ""); }
        },

        _request: function (sMethod, sPath, oBody) {
            var that = this;
            return new Promise(function (resolve, reject) {
                jQuery.ajax({
                    url: that._base + sPath,
                    method: sMethod,
                    contentType: "application/json",
                    dataType: "json",
                    data: oBody ? JSON.stringify(oBody) : undefined,
                    success: function (oData) { resolve(oData); },
                    error: function (oXhr) {
                        var sMsg = "Request failed (" + oXhr.status + ")";
                        try {
                            var oErr = JSON.parse(oXhr.responseText);
                            if (oErr && oErr.message) { sMsg = oErr.message; }
                        } catch (e) { /* keep default message */ }
                        reject(new Error(sMsg));
                    }
                });
            });
        },

        // ---- Employees -----------------------------------------------------
        getEmployees: function (sSearch, sKeyDate) {
            var aParams = [];
            if (sSearch) { aParams.push("search=" + encodeURIComponent(sSearch)); }
            if (sKeyDate) { aParams.push("keyDate=" + sKeyDate); }
            var sQuery = aParams.length ? "?" + aParams.join("&") : "";
            return this._request("GET", "/employees" + sQuery);
        },

        getEmployee: function (iPernr, sKeyDate) {
            var sQuery = sKeyDate ? "?keyDate=" + sKeyDate : "";
            return this._request("GET", "/employees/" + iPernr + sQuery);
        },

        hireEmployee: function (oData) {
            return this._request("POST", "/employees/hire", oData);
        },

        updatePersonalData: function (iPernr, oData) {
            return this._request("PUT", "/employees/" + iPernr + "/personaldata", oData);
        },

        reassign: function (iPernr, oData) {
            return this._request("PUT", "/employees/" + iPernr + "/reassign", oData);
        },

        getLeaveBalances: function (iPernr) {
            return this._request("GET", "/employees/" + iPernr + "/leave-balances");
        },

        recordAbsence: function (iPernr, oData) {
            return this._request("POST", "/employees/" + iPernr + "/absences", oData);
        },

        // ---- Organizational Management -------------------------------------
        getOrgUnits: function (sKeyDate) {
            var sQuery = sKeyDate ? "?keyDate=" + sKeyDate : "";
            return this._request("GET", "/orgunits" + sQuery);
        },

        getOrgStructure: function (iRoot, sKeyDate) {
            var sQuery = sKeyDate ? "?keyDate=" + sKeyDate : "";
            return this._request("GET", "/orgunits/" + iRoot + "/structure" + sQuery);
        },

        getPositions: function (iOrgUnit) {
            var sQuery = iOrgUnit ? "?orgUnitId=" + iOrgUnit : "";
            return this._request("GET", "/orgunits/positions" + sQuery);
        },

        // ---- Value help ----------------------------------------------------
        getValueHelp: function (sName) {
            return this._request("GET", "/valuehelp/" + sName);
        },

        getDomain: function (sDomain) {
            return this._request("GET", "/valuehelp/domain/" + sDomain);
        }
    };
});
