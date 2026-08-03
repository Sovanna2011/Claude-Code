sap.ui.define([], function () {
    "use strict";

    return {

        /**
         * Maps a release indicator (FRGKE) to a semantic state.
         * @param {string} sFrgke ' ' / 'B' / 'R'
         * @returns {sap.ui.core.ValueState} state
         */
        releaseState: function (sFrgke) {
            switch (sFrgke) {
                case "R": return "Success";
                case "B": return "Warning";
                default:  return "None";
            }
        },

        /**
         * Icon for a release step depending on whether it is released.
         * @param {boolean} bReleased released flag
         * @returns {string} icon uri
         */
        stepIcon: function (bReleased) {
            return bReleased ? "sap-icon://accept" : "sap-icon://pending";
        },

        stepState: function (bReleased) {
            return bReleased ? "Success" : "Warning";
        },

        /**
         * Formats an amount as a grouped decimal (currency is shown separately
         * via the control's unit), e.g. 12500 -> "12,500.00".
         * @param {number} nValue amount
         * @returns {string} formatted number
         */
        amount: function (nValue) {
            if (nValue === undefined || nValue === null) { return ""; }
            var oFormat = sap.ui.core.format.NumberFormat.getFloatInstance({
                minFractionDigits: 2,
                maxFractionDigits: 2,
                groupingEnabled: true
            });
            return oFormat.format(nValue);
        }
    };
});
