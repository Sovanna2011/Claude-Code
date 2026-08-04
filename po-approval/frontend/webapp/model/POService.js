sap.ui.define([], function () {
    "use strict";

    /**
     * Thin wrapper over the PO Approval REST API. All methods return Promises
     * resolving to the parsed JSON payload (or rejecting with an Error whose
     * message is the API error text). A bearer token, once set via setToken,
     * is sent on every request for user-level authentication.
     */
    return {

        _base: "/api",
        _token: null,

        /**
         * Sets the API base URL (from the config model).
         * @param {string} sBase base url, e.g. "/api"
         */
        setBase: function (sBase) {
            if (sBase) { this._base = sBase.replace(/\/$/, ""); }
        },

        /**
         * Sets (or clears) the bearer token used for authentication.
         * @param {string|null} sToken the token, or null to clear
         */
        setToken: function (sToken) {
            this._token = sToken || null;
        },

        _request: function (sMethod, sPath, oBody) {
            var that = this;
            return new Promise(function (resolve, reject) {
                jQuery.ajax({
                    url: that._base + sPath,
                    method: sMethod,
                    contentType: "application/json",
                    dataType: "json",
                    beforeSend: function (oXhr) {
                        if (that._token) {
                            oXhr.setRequestHeader("Authorization", "Bearer " + that._token);
                        }
                    },
                    data: oBody ? JSON.stringify(oBody) : undefined,
                    success: function (oData) { resolve(oData); },
                    error: function (oXhr) {
                        var sMsg = "Request failed (" + oXhr.status + ")";
                        try {
                            var oErr = JSON.parse(oXhr.responseText);
                            if (oErr && oErr.message) { sMsg = oErr.message; }
                        } catch (e) { /* keep default message */ }
                        var oError = new Error(sMsg);
                        oError.status = oXhr.status;
                        reject(oError);
                    }
                });
            });
        },

        // ---- Authentication ------------------------------------------------
        login: function (sUsername, sPassword) {
            return this._request("POST", "/auth/login", { username: sUsername, password: sPassword });
        },

        getMe: function () {
            return this._request("GET", "/auth/me");
        },

        // ---- Purchase orders -----------------------------------------------
        getWorklist: function (sSearch, bOnlyPending, sGroup) {
            var aParams = [];
            if (sSearch) { aParams.push("search=" + encodeURIComponent(sSearch)); }
            if (bOnlyPending) { aParams.push("onlyPending=true"); }
            if (sGroup) { aParams.push("purchasingGroup=" + encodeURIComponent(sGroup)); }
            var sQuery = aParams.length ? "?" + aParams.join("&") : "";
            return this._request("GET", "/purchase-orders" + sQuery);
        },

        getPendingByStrategy: function (bAssignedToMe) {
            var sQuery = bAssignedToMe ? "?assignedToMe=true" : "";
            return this._request("GET", "/purchase-orders/pending-by-strategy" + sQuery);
        },

        getPurchaseOrder: function (sEbeln) {
            return this._request("GET", "/purchase-orders/" + sEbeln);
        },

        release: function (sEbeln, oData) {
            return this._request("POST", "/purchase-orders/" + sEbeln + "/release", oData || {});
        },

        reject: function (sEbeln, oData) {
            return this._request("POST", "/purchase-orders/" + sEbeln + "/reject", oData);
        },

        // ---- Value help ----------------------------------------------------
        getValueHelp: function (sName) {
            return this._request("GET", "/valuehelp/" + sName);
        }
    };
});
