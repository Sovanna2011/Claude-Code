sap.ui.define([], function () {
	"use strict";

	/**
	 * Formatters render stored values for display.
	 *
	 * The rule the whole application follows: values are stored and transported
	 * as exact decimals and ISO dates, and are made business-friendly only at
	 * the moment they are shown. Nothing here is ever fed back into a
	 * calculation.
	 */
	var formatter = {

		/** tons formats a tonnage with thousands separators and three decimals. */
		tons: function (vValue) {
			return formatter._number(vValue, 3);
		},

		/** tons0 formats a tonnage rounded to whole tons, for headline tiles. */
		tons0: function (vValue) {
			return formatter._number(vValue, 0);
		},

		/** percent formats a percentage to two decimals with its sign. */
		percent: function (vValue) {
			var sNumber = formatter._number(vValue, 2);
			return sNumber === "" ? "" : sNumber + " %";
		},

		/**
		 * money renders a money amount short enough for a KPI tile.
		 *
		 * A season's cost runs to tens of millions, and NumericContent truncates
		 * what does not fit rather than abbreviating it - 76,606,286 arrives on
		 * screen as "76,6", which reads as seventy-six point six of something.
		 * Abbreviating here keeps the magnitude legible. Tables keep the full
		 * figure, because that is where somebody adds the column up.
		 */
		money: function (vValue) {
			if (vValue === null || vValue === undefined || vValue === "") {
				return "";
			}
			var fValue = typeof vValue === "number" ? vValue : parseFloat(vValue);
			if (isNaN(fValue)) {
				return String(vValue);
			}
			var fAbs = Math.abs(fValue);
			if (fAbs >= 1e9) {
				return formatter._number(fValue / 1e9, 2) + "bn";
			}
			if (fAbs >= 1e6) {
				return formatter._number(fValue / 1e6, 1) + "m";
			}
			if (fAbs >= 1e4) {
				return formatter._number(fValue / 1e3, 0) + "k";
			}
			return formatter._number(fValue, 0);
		},

		/** units formats a whole count, such as a number of bags. */
		units: function (vValue) {
			return formatter._number(vValue, 0);
		},

		_number: function (vValue, iDecimals) {
			if (vValue === null || vValue === undefined || vValue === "") {
				return "";
			}
			var fValue = typeof vValue === "number" ? vValue : parseFloat(vValue);
			if (isNaN(fValue)) {
				return String(vValue);
			}
			return fValue.toLocaleString(sap.ui.getCore().getConfiguration().getLanguage(), {
				minimumFractionDigits: iDecimals,
				maximumFractionDigits: iDecimals
			});
		},

		/** date renders an ISO business date in the user's locale. */
		date: function (sValue) {
			if (!sValue) {
				return "";
			}
			var oDate = new Date(sValue + "T00:00:00Z");
			if (isNaN(oDate.getTime())) {
				return sValue;
			}
			return oDate.toLocaleDateString(undefined, {
				year: "numeric", month: "short", day: "2-digit", timeZone: "UTC"
			});
		},

		/** dateTime renders a stored UTC instant. */
		dateTime: function (sValue) {
			if (!sValue) {
				return "";
			}
			var oDate = new Date(sValue);
			return isNaN(oDate.getTime()) ? sValue : oDate.toLocaleString();
		},

		// ------------------------------------------------------------------
		// Semantic colouring
		// ------------------------------------------------------------------

		/** severityState maps a domain severity onto a Fiori value state, so
		 * the same four colours mean the same four things everywhere. */
		severityState: function (sSeverity) {
			switch (sSeverity) {
				case "ERROR": return "Error";
				case "WARNING": return "Warning";
				case "SUCCESS": return "Success";
				default: return "Information";
			}
		},

		/**
		 * severityColor maps a severity onto sap.m.ValueColor.
		 *
		 * NumericContent and the micro charts take ValueColor
		 * (Good/Critical/Error/Neutral), which is a different enumeration from
		 * the ValueState used by ObjectStatus and ProgressIndicator. Mixing the
		 * two throws at render time, so each has its own formatter.
		 */
		severityColor: function (sSeverity) {
			switch (sSeverity) {
				case "ERROR": return "Error";
				case "WARNING": return "Critical";
				case "SUCCESS": return "Good";
				default: return "Neutral";
			}
		},

		/** achievementColor grades a percentage against plan, as a ValueColor. */
		achievementColor: function (vPct) {
			var fPct = parseFloat(vPct);
			if (isNaN(fPct) || fPct === 0) {
				return "Neutral";
			}
			if (fPct >= 98) { return "Good"; }
			if (fPct >= 90) { return "Critical"; }
			return "Error";
		},

		/** daysColor colours a schedule slip, as a ValueColor. */
		daysColor: function (iDays) {
			if (!iDays || iDays <= 0) {
				return "Good";
			}
			return iDays > 7 ? "Error" : "Critical";
		},

		severityIcon: function (sSeverity) {
			switch (sSeverity) {
				case "ERROR": return "sap-icon://error";
				case "WARNING": return "sap-icon://alert";
				case "SUCCESS": return "sap-icon://sys-enter-2";
				default: return "sap-icon://information";
			}
		},

		/** statusState colours a plan version status. */
		statusState: function (sStatus) {
			switch (sStatus) {
				case "RELEASED": return "Success";
				case "APPROVED": return "Success";
				case "IN_REVIEW": return "Warning";
				case "REJECTED": return "Error";
				case "SUPERSEDED":
				case "CLOSED": return "None";
				default: return "Information";
			}
		},

		/** capacityState turns a capacity percentage into a warning colour.
		 * The thresholds mirror the server-side defaults. */
		capacityState: function (vPct) {
			var fPct = parseFloat(vPct);
			if (isNaN(fPct)) {
				return "None";
			}
			if (fPct >= 100) { return "Error"; }
			if (fPct >= 90) { return "Error"; }
			if (fPct >= 80) { return "Warning"; }
			return "Success";
		},

		/** varianceState colours a variance: below plan is a warning, at or
		 * above plan is a success. */
		varianceState: function (vValue) {
			var fValue = parseFloat(vValue);
			if (isNaN(fValue) || fValue === 0) {
				return "None";
			}
			return fValue < 0 ? "Warning" : "Success";
		},

		/** achievementState grades a percentage against plan. */
		achievementState: function (vPct) {
			var fPct = parseFloat(vPct);
			if (isNaN(fPct) || fPct === 0) {
				return "None";
			}
			if (fPct >= 98) { return "Success"; }
			if (fPct >= 90) { return "Warning"; }
			return "Error";
		},

		/** statusText turns an enum into readable text. */
		statusText: function (sStatus) {
			if (!sStatus) {
				return "";
			}
			return sStatus.charAt(0) + sStatus.slice(1).toLowerCase().replace(/_/g, " ");
		},

		/** actionText labels a workflow button. */
		actionText: function (sAction) {
			var mLabels = {
				SUBMIT: "Submit for review",
				RECALL: "Recall",
				APPROVE: "Approve",
				REJECT: "Reject",
				RELEASE: "Release",
				SUPERSEDE: "Supersede",
				CLOSE: "Close",
				REOPEN: "Reopen"
			};
			return mLabels[sAction] || sAction;
		},

		/** daysLabel renders a schedule slip. */
		daysLabel: function (iDays) {
			if (!iDays) {
				return "On schedule";
			}
			var iAbs = Math.abs(iDays);
			var sUnit = iAbs === 1 ? "day" : "days";
			return iDays > 0 ? iAbs + " " + sUnit + " late" : iAbs + " " + sUnit + " early";
		},

		daysState: function (iDays) {
			if (!iDays || iDays <= 0) {
				return "Success";
			}
			return iDays > 7 ? "Error" : "Warning";
		},

		// ------------------------------------------------------------------
		// Execution
		// ------------------------------------------------------------------

		/** orderState colours an order by how far through its life it is. */
		orderState: function (sStatus) {
			switch (sStatus) {
				case "COMPLETED":
				case "TECHNICALLY_CLOSED": return "Success";
				case "CANCELLED": return "Error";
				case "PARTIALLY_CONFIRMED":
				case "IN_PROCESS": return "Warning";
				default: return "Information";
			}
		},

		/** orderStatusLabel turns the stored status into a readable phrase. */
		orderStatusLabel: function (sStatus) {
			var mLabels = {
				PLANNED: "Planned",
				RELEASED: "Released",
				IN_PROCESS: "In process",
				PARTIALLY_CONFIRMED: "Partly confirmed",
				COMPLETED: "Completed",
				TECHNICALLY_CLOSED: "Closed",
				CANCELLED: "Cancelled"
			};
			return mLabels[sStatus] || sStatus;
		},

		/** qualityState colours a laboratory verdict. */
		qualityState: function (sStatus) {
			switch (sStatus) {
				case "FAIL": return "Error";
				case "WARNING": return "Warning";
				case "PASS": return "Success";
				default: return "None";
			}
		},

		/** docTypeLabel names a stock movement in the words a keeper uses. */
		docTypeLabel: function (sType) {
			var mLabels = {
				RECEIPT: "Goods receipt",
				ISSUE: "Goods issue",
				TRANSFER: "Transfer",
				ADJUSTMENT: "Adjustment",
				COUNT: "Stock count",
				HOLD: "Quality hold",
				RELEASE: "Hold released",
				SHIPMENT: "Shipment",
				REVERSAL: "Reversal"
			};
			return mLabels[sType] || sType;
		},

		/** utilisationState warns before a store is full rather than after. */
		utilisationState: function (vPercent) {
			var fValue = parseFloat(vPercent);
			if (isNaN(fValue)) {
				return "None";
			}
			if (fValue >= 100) {
				return "Error";
			}
			return fValue >= 90 ? "Warning" : "Success";
		},

		/** heldState draws attention to a store holding blocked stock. */
		heldState: function (vHeld) {
			return parseFloat(vHeld) > 0 ? "Warning" : "None";
		},

		/** holdStatus says in one word whether a hold is still blocking. */
		holdStatus: function (sReleasedOn) {
			return sReleasedOn ? "Released" : "Blocking";
		},

		holdState: function (sReleasedOn) {
			return sReleasedOn ? "Success" : "Warning";
		},

		/** maintenanceState colours a window by whether it has been approved,
		 * which is what decides if it costs the plan a crushing day. */
		maintenanceState: function (sStatus) {
			switch (sStatus) {
				case "APPROVED": return "Warning";
				case "DONE": return "Success";
				case "CANCELLED": return "Error";
				default: return "Information";
			}
		},

		/** varianceColor colours a money variance on a tile. Spending more than
		 * the plan is bad news whatever the sign convention says. */
		varianceColor: function (vValue) {
			var fValue = parseFloat(vValue);
			if (isNaN(fValue) || fValue === 0) {
				return "Neutral";
			}
			return fValue > 0 ? "Error" : "Good";
		},

		/** reversedLabel marks a document that has been undone. */
		reversedLabel: function (bReversed) {
			return bReversed ? "Reversed" : "";
		}
	};

	return formatter;
});
