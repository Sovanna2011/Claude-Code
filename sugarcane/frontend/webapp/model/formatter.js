sap.ui.define([], function () {
	"use strict";

	// Hectares everywhere, to two decimals, with thousands separators: the specification's unit,
	// formatted once so no screen invents its own.
	var HA = new Intl.NumberFormat("en-GB", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
	var HA_SHORT = new Intl.NumberFormat("en-GB", { maximumFractionDigits: 0 });

	return {
		hectares: function (value) {
			return typeof value === "number" ? HA.format(value) : "";
		},

		hectaresUnit: function (value) {
			return typeof value === "number" ? HA.format(value) + " ha" : "—";
		},

		hectaresShort: function (value) {
			return typeof value === "number" ? HA_SHORT.format(value) : "";
		},

		signedHectares: function (value) {
			if (typeof value !== "number") {
				return "";
			}
			return (value > 0 ? "+" : "") + HA.format(value);
		},

		percent: function (value) {
			return typeof value === "number" ? value.toFixed(1) + "%" : "";
		},

		percentOfTotal: function (value) {
			return typeof value === "number" ? value.toFixed(1) + "% of total area" : "";
		},

		nodeIcon: function (nodeType) {
			switch (nodeType) {
				case "Farm": return "sap-icon://building";
				case "Zone": return "sap-icon://map-2";
				default: return "sap-icon://tree";
			}
		},

		kpiIcon: function (key) {
			switch (key) {
				case "total": return "sap-icon://map";
				case "newPlanting": return "sap-icon://sys-add";
				case "ratoon": return "sap-icon://refresh";
				case "withCane": return "sap-icon://accept";
				case "available": return "sap-icon://open-folder";
				default: return "sap-icon://blocked";
			}
		},

		// The colour says what the figure means, not merely that it is large: land under cane is
		// good news, land that cannot be planted is not, and the total is neither.
		kpiColour: function (key) {
			switch (key) {
				case "withCane": return "Good";
				case "available": return "Critical";
				case "cannotPlant": return "Error";
				default: return "Neutral";
			}
		},

		coverageState: function (percent) {
			if (typeof percent !== "number") {
				return "None";
			}
			if (percent >= 60) {
				return "Success";
			}
			return percent >= 25 ? "Warning" : "None";
		},

		varianceState: function (value) {
			if (typeof value !== "number" || Math.abs(value) < 0.005) {
				return "None";
			}
			return value < 0 ? "Error" : "Success";
		},

		achievementState: function (percent) {
			if (typeof percent !== "number") {
				return "None";
			}
			if (percent >= 95) {
				return "Success";
			}
			return percent >= 80 ? "Warning" : "Error";
		},

		// The stored boundary and the registered area are two measurements of the same field. When
		// they disagree by more than a rounding, the survey and the paperwork have drifted apart
		// and someone should look.
		boundaryText: function (areaHa) {
			return typeof areaHa === "number"
				? "Polygon stored · " + HA.format(areaHa) + " ha measured"
				: "No boundary stored";
		},

		boundaryState: function (areaHa, totalHa) {
			if (typeof areaHa !== "number") {
				return "None";
			}
			if (typeof totalHa !== "number" || totalHa <= 0) {
				return "Information";
			}
			return Math.abs(areaHa - totalHa) / totalHa <= 0.02 ? "Success" : "Warning";
		},

		remainderState: function (value) {
			if (typeof value !== "number") {
				return "None";
			}
			return value < 0 ? "Error" : "None";
		},

		reasonCoverage: function (recorded, capacity) {
			var r = typeof recorded === "number" ? recorded : 0;
			var c = typeof capacity === "number" ? capacity : 0;
			return HA.format(r) + " of " + HA.format(c) + " ha accounted for";
		},

		reasonCoverageState: function (recorded, capacity) {
			var r = typeof recorded === "number" ? recorded : 0;
			var c = typeof capacity === "number" ? capacity : 0;
			if (r > c + 0.0001) {
				return "Error";
			}
			return Math.abs(r - c) < 0.0001 ? "Success" : "Warning";
		},

		withinPlantableState: function (withCane, plantable) {
			if (typeof withCane !== "number" || typeof plantable !== "number") {
				return "None";
			}
			return withCane > plantable + 0.0001 ? "Error" : "Success";
		}
	};
});
