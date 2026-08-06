sap.ui.define([], function () {
	"use strict";

	/**
	 * chart draws the dashboard curves as inline SVG.
	 *
	 * A charting library would be a large dependency for two line charts, and
	 * an on-premises deployment has to serve every asset itself. SVG generated
	 * here scales cleanly, prints, and inherits the theme's text colour.
	 *
	 * The colours are the Fiori qualitative palette so that "target" and
	 * "actual" mean the same thing on every page.
	 */
	var COLOUR_TARGET = "#5899DA";
	var COLOUR_ACTUAL = "#2B7D2B";
	var COLOUR_AXIS = "#8c8c8c";

	function escapeHtml(sValue) {
		return String(sValue === undefined || sValue === null ? "" : sValue)
			.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")
			.replace(/"/g, "&quot;");
	}

	function toNumber(vValue) {
		var fValue = typeof vValue === "number" ? vValue : parseFloat(vValue);
		return isNaN(fValue) ? 0 : fValue;
	}

	function formatTons(fValue) {
		if (Math.abs(fValue) >= 1000000) {
			return (fValue / 1000000).toFixed(2) + " Mt";
		}
		if (Math.abs(fValue) >= 1000) {
			return Math.round(fValue / 1000) + " kt";
		}
		return Math.round(fValue) + " t";
	}

	// One colour per cost category, held here so the bar chart and any future
	// legend cannot drift apart.
	var CATEGORY_COLOURS = {
		CANE: "#5899DA", LABOUR: "#E8743B", ENERGY: "#19A979",
		CHEMICALS: "#945ECF", PACKAGING: "#13A4B4", MAINTENANCE: "#BF399E",
		OVERHEAD: "#8c8c8c"
	};

	return {

		/**
		 * cumulativeCurve renders cumulative target against cumulative actual.
		 *
		 * The actual line stops at the last day that carries a recorded actual,
		 * rather than running flat to the end of the season, because a flat line
		 * to December reads as "we stopped crushing" instead of "we have not got
		 * there yet".
		 */
		cumulativeCurve: function (aPoints) {
			if (!aPoints || !aPoints.length) {
				return "<div class='sugarChartEmpty'>No plan data for this season yet.</div>";
			}

			var iWidth = 960, iHeight = 280;
			var iLeft = 70, iRight = 20, iTop = 16, iBottom = 34;
			var iPlotW = iWidth - iLeft - iRight;
			var iPlotH = iHeight - iTop - iBottom;

			var fMax = 0;
			aPoints.forEach(function (oPoint) {
				fMax = Math.max(fMax, toNumber(oPoint.cumTarget), toNumber(oPoint.cumActual));
			});
			if (fMax <= 0) {
				fMax = 1;
			}

			var iCount = aPoints.length;
			function x(i) {
				return iLeft + (iCount <= 1 ? 0 : (iPlotW * i) / (iCount - 1));
			}
			function y(fValue) {
				return iTop + iPlotH - (iPlotH * fValue) / fMax;
			}

			var aTargetPath = [];
			var aActualPath = [];
			var iLastActual = -1;
			aPoints.forEach(function (oPoint, i) {
				aTargetPath.push((i === 0 ? "M" : "L") + x(i).toFixed(1) + " " + y(toNumber(oPoint.cumTarget)).toFixed(1));
				if (oPoint.hasActual) {
					aActualPath.push((iLastActual < 0 ? "M" : "L") +
						x(i).toFixed(1) + " " + y(toNumber(oPoint.cumActual)).toFixed(1));
					iLastActual = i;
				}
			});

			// Horizontal grid at quarters of the maximum.
			var aGrid = [];
			for (var g = 0; g <= 4; g++) {
				var fValue = (fMax * g) / 4;
				var fY = y(fValue);
				aGrid.push("<line x1='" + iLeft + "' y1='" + fY.toFixed(1) +
					"' x2='" + (iWidth - iRight) + "' y2='" + fY.toFixed(1) +
					"' stroke='" + COLOUR_AXIS + "' stroke-opacity='0.25' stroke-width='1'/>");
				aGrid.push("<text x='" + (iLeft - 8) + "' y='" + (fY + 4).toFixed(1) +
					"' text-anchor='end' font-size='11' fill='" + COLOUR_AXIS + "'>" +
					escapeHtml(formatTons(fValue)) + "</text>");
			}

			// Date labels at the ends and the middle: a 137-day axis cannot
			// carry a label per day.
			var aLabels = [];
			[0, Math.floor(iCount / 2), iCount - 1].forEach(function (i, iIndex) {
				if (i < 0 || i >= iCount) {
					return;
				}
				var sAnchor = iIndex === 0 ? "start" : (iIndex === 2 ? "end" : "middle");
				aLabels.push("<text x='" + x(i).toFixed(1) + "' y='" + (iHeight - 12) +
					"' text-anchor='" + sAnchor + "' font-size='11' fill='" + COLOUR_AXIS + "'>" +
					escapeHtml(aPoints[i].date) + "</text>");
			});

			// A marker where the actual line stops, labelled with the position.
			var sMarker = "";
			if (iLastActual >= 0) {
				var oLast = aPoints[iLastActual];
				sMarker = "<circle cx='" + x(iLastActual).toFixed(1) + "' cy='" +
					y(toNumber(oLast.cumActual)).toFixed(1) + "' r='4' fill='" + COLOUR_ACTUAL + "'/>" +
					"<text x='" + (x(iLastActual) + 8).toFixed(1) + "' y='" +
					(y(toNumber(oLast.cumActual)) - 8).toFixed(1) +
					"' font-size='11' fill='" + COLOUR_ACTUAL + "'>" +
					escapeHtml(formatTons(toNumber(oLast.cumActual)) + " on " + oLast.date) + "</text>";
			}

			return "<div class='sugarChart' role='img' aria-label='" +
				escapeHtml("Cumulative cane target against actual, " + iCount + " days") + "'>" +
				"<svg viewBox='0 0 " + iWidth + " " + iHeight + "' width='100%' height='" + iHeight +
				"' preserveAspectRatio='xMidYMid meet'>" +
				aGrid.join("") +
				"<line x1='" + iLeft + "' y1='" + iTop + "' x2='" + iLeft + "' y2='" + (iTop + iPlotH) +
				"' stroke='" + COLOUR_AXIS + "' stroke-width='1'/>" +
				"<line x1='" + iLeft + "' y1='" + (iTop + iPlotH) + "' x2='" + (iWidth - iRight) +
				"' y2='" + (iTop + iPlotH) + "' stroke='" + COLOUR_AXIS + "' stroke-width='1'/>" +
				"<path d='" + aTargetPath.join(" ") + "' fill='none' stroke='" + COLOUR_TARGET +
				"' stroke-width='2' stroke-dasharray='6 4'/>" +
				(aActualPath.length
					? "<path d='" + aActualPath.join(" ") + "' fill='none' stroke='" + COLOUR_ACTUAL +
					"' stroke-width='2.5'/>"
					: "") +
				sMarker +
				aLabels.join("") +
				"</svg></div>";
		},

		/**
		 * stockCurve renders a warehouse balance against its capacity lines, so
		 * the day the store fills is visible rather than calculated.
		 */
		stockCurve: function (aDays, fUsable, fWarnPct) {
			if (!aDays || !aDays.length) {
				return "<div class='sugarChartEmpty'>No stock movements in this period.</div>";
			}
			var iWidth = 960, iHeight = 240;
			var iLeft = 70, iRight = 20, iTop = 16, iBottom = 34;
			var iPlotW = iWidth - iLeft - iRight;
			var iPlotH = iHeight - iTop - iBottom;

			var fMax = toNumber(fUsable);
			aDays.forEach(function (oDay) {
				fMax = Math.max(fMax, toNumber(oDay.endingBalance));
			});
			fMax = fMax * 1.05 || 1;

			var iCount = aDays.length;
			function x(i) {
				return iLeft + (iCount <= 1 ? 0 : (iPlotW * i) / (iCount - 1));
			}
			function y(fValue) {
				return iTop + iPlotH - (iPlotH * fValue) / fMax;
			}

			var aPath = aDays.map(function (oDay, i) {
				return (i === 0 ? "M" : "L") + x(i).toFixed(1) + " " +
					y(toNumber(oDay.endingBalance)).toFixed(1);
			});

			var fWarnLine = toNumber(fUsable) * (toNumber(fWarnPct) || 80) / 100;
			var sCapacity =
				"<line x1='" + iLeft + "' y1='" + y(toNumber(fUsable)).toFixed(1) +
				"' x2='" + (iWidth - iRight) + "' y2='" + y(toNumber(fUsable)).toFixed(1) +
				"' stroke='#BB0000' stroke-width='2' stroke-dasharray='8 4'/>" +
				"<text x='" + (iWidth - iRight) + "' y='" + (y(toNumber(fUsable)) - 6).toFixed(1) +
				"' text-anchor='end' font-size='11' fill='#BB0000'>Usable capacity " +
				escapeHtml(formatTons(toNumber(fUsable))) + "</text>" +
				"<line x1='" + iLeft + "' y1='" + y(fWarnLine).toFixed(1) +
				"' x2='" + (iWidth - iRight) + "' y2='" + y(fWarnLine).toFixed(1) +
				"' stroke='#E76500' stroke-width='1.5' stroke-dasharray='4 4'/>";

			var aLabels = [];
			[0, Math.floor(iCount / 2), iCount - 1].forEach(function (i, iIndex) {
				if (i < 0 || i >= iCount) {
					return;
				}
				var sAnchor = iIndex === 0 ? "start" : (iIndex === 2 ? "end" : "middle");
				aLabels.push("<text x='" + x(i).toFixed(1) + "' y='" + (iHeight - 12) +
					"' text-anchor='" + sAnchor + "' font-size='11' fill='" + COLOUR_AXIS + "'>" +
					escapeHtml(aDays[i].date) + "</text>");
			});

			return "<div class='sugarChart' role='img' aria-label='Warehouse stock against capacity'>" +
				"<svg viewBox='0 0 " + iWidth + " " + iHeight + "' width='100%' height='" + iHeight + "'>" +
				"<line x1='" + iLeft + "' y1='" + (iTop + iPlotH) + "' x2='" + (iWidth - iRight) +
				"' y2='" + (iTop + iPlotH) + "' stroke='" + COLOUR_AXIS + "' stroke-width='1'/>" +
				sCapacity +
				"<path d='" + aPath.join(" ") + "' fill='none' stroke='" + COLOUR_TARGET + "' stroke-width='2.5'/>" +
				aLabels.join("") +
				"</svg></div>";
		},

		/**
		 * costByCategory draws the planned and actual cost of each category
		 * side by side.
		 *
		 * A category rather than an element: twelve elements is too many bars to
		 * read, and the question the chart answers - where did the money go -
		 * is a category question. The table underneath carries the detail.
		 */
		costByCategory: function (aLines) {
			if (!aLines || !aLines.length) {
				return "";
			}

			var mByCategory = {};
			aLines.forEach(function (oLine) {
				var sKey = oLine.category || "OVERHEAD";
				if (!mByCategory[sKey]) {
					mByCategory[sKey] = { planned: 0, actual: 0 };
				}
				mByCategory[sKey].planned += toNumber(oLine.plannedCost);
				mByCategory[sKey].actual += toNumber(oLine.actualCost);
			});

			var aKeys = Object.keys(mByCategory).sort();
			var fMax = 0;
			aKeys.forEach(function (sKey) {
				fMax = Math.max(fMax, mByCategory[sKey].planned, mByCategory[sKey].actual);
			});
			if (fMax <= 0) {
				return "";
			}

			var iWidth = 760, iHeight = 260, iLeft = 90, iRight = 20, iTop = 16, iBottom = 46;
			var iPlotW = iWidth - iLeft - iRight;
			var iPlotH = iHeight - iTop - iBottom;
			var fSlot = iPlotW / aKeys.length;
			var fBar = Math.min(26, fSlot / 3);

			var aBars = [];
			aKeys.forEach(function (sKey, i) {
				var oEntry = mByCategory[sKey];
				var sColour = CATEGORY_COLOURS[sKey] || COLOUR_AXIS;
				var fCentre = iLeft + fSlot * (i + 0.5);

				[["planned", oEntry.planned, 0.35], ["actual", oEntry.actual, 1]].forEach(
					function (aBarSpec, iBar) {
						var fValue = aBarSpec[1];
						var fH = (fValue / fMax) * iPlotH;
						var fX = fCentre + (iBar === 0 ? -fBar - 2 : 2);
						aBars.push("<rect x='" + fX.toFixed(1) + "' y='" +
							(iTop + iPlotH - fH).toFixed(1) + "' width='" + fBar.toFixed(1) +
							"' height='" + Math.max(0, fH).toFixed(1) + "' fill='" + sColour +
							"' fill-opacity='" + aBarSpec[2] + "'><title>" +
							escapeHtml(sKey + " " + aBarSpec[0] + ": " +
								fValue.toLocaleString(undefined, { maximumFractionDigits: 0 })) +
							"</title></rect>");
					});

				aBars.push("<text x='" + fCentre.toFixed(1) + "' y='" + (iHeight - 26) +
					"' text-anchor='middle' font-size='10' fill='" + COLOUR_AXIS + "'>" +
					escapeHtml(sKey.slice(0, 9)) + "</text>");
			});

			return "<div class='sugarChart' role='img' " +
				"aria-label='Planned and actual cost by category'>" +
				"<svg viewBox='0 0 " + iWidth + " " + iHeight + "' width='100%' height='" +
				iHeight + "'>" +
				"<line x1='" + iLeft + "' y1='" + (iTop + iPlotH) + "' x2='" + (iWidth - iRight) +
				"' y2='" + (iTop + iPlotH) + "' stroke='" + COLOUR_AXIS + "' stroke-width='1'/>" +
				aBars.join("") +
				"<text x='" + iLeft + "' y='" + (iHeight - 8) + "' font-size='10' fill='" +
				COLOUR_AXIS + "'>Left bar planned, right bar actual</text>" +
				"</svg></div>";
		}
	};
});
