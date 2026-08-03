sap.ui.define([
    "sap/ui/core/format/DateFormat"
], function (DateFormat) {
    "use strict";

    var oDateFmt = DateFormat.getDateInstance({ pattern: "dd.MM.yyyy" });

    return {

        /**
         * Formats an ISO date string as dd.MM.yyyy. The SAP high date
         * (9999-12-31) is rendered as "unlimited".
         * @param {string} sIso ISO date
         * @returns {string} formatted date
         */
        date: function (sIso) {
            if (!sIso) { return ""; }
            if (sIso.indexOf("9999") === 0) { return "unlimited"; }
            var oDate = new Date(sIso);
            return isNaN(oDate.getTime()) ? sIso : oDateFmt.format(oDate);
        },

        /**
         * Formats an amount with currency, e.g. "96,000.00 EUR".
         * @param {number} fAmount amount
         * @param {string} sCurrency ISO currency code
         * @returns {string} formatted amount
         */
        amount: function (fAmount, sCurrency) {
            if (fAmount === null || fAmount === undefined) { return ""; }
            var sNum = Number(fAmount).toLocaleString("en-US", {
                minimumFractionDigits: 2, maximumFractionDigits: 2
            });
            return sCurrency ? sNum + " " + sCurrency : sNum;
        },

        /**
         * Maps the SAP employment status text to a sap.ui.core.ValueState.
         * @param {string} sStatus status text
         * @returns {sap.ui.core.ValueState} value state
         */
        statusState: function (sStatus) {
            switch (sStatus) {
                case "Active": return "Success";
                case "Inactive": return "Warning";
                case "Withdrawn": return "Error";
                default: return "None";
            }
        },

        /**
         * Vacancy state for a position (true = vacant -> Warning).
         * @param {boolean} bVacant vacancy flag
         * @returns {sap.ui.core.ValueState} value state
         */
        vacantState: function (bVacant) {
            return bVacant ? "Warning" : "Success";
        },

        /**
         * Vacancy text.
         * @param {boolean} bVacant vacancy flag
         * @returns {string} text
         */
        vacantText: function (bVacant) {
            return bVacant ? "Vacant" : "Occupied";
        }
    };
});
