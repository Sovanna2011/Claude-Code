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
	// Short of target and the rolling average. They are named rather than
	// inlined so that "behind plan" is the same red on every chart.
	var COLOUR_SHORT = "#BB0000";
	var COLOUR_ROLLING = "#E76500";

	// The qualitative palette for charts with one band per product or channel.
	// Chosen to stay distinguishable in the common forms of colour blindness;
	// every band also carries a <title>, so the chart is readable without
	// relying on colour at all.
	var BAND_COLOURS = [
		"#5899DA", "#E8743B", "#19A979", "#945ECF", "#13A4B4",
		"#BF399E", "#EE6868", "#6C8893"
	];

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

	/**
	 * gridLines draws the horizontal rules and their value labels.
	 *
	 * Every chart here needs them and they were being written out four times,
	 * which is four places for the label formatting to drift apart.
	 *
	 * fMin is the bottom of the axis, which is zero for a tonnage chart and the
	 * bottom of the operating range for a recovery one.
	 */
	function gridLines(iLeft, iRight, fnY, fMax, fnFormat, fMin) {
		var fBottom = fMin || 0;
		var aOut = [];
		for (var g = 0; g <= 4; g++) {
			var fValue = fBottom + ((fMax - fBottom) * g) / 4;
			var fY = fnY(fValue);
			aOut.push("<line x1='" + iLeft + "' y1='" + fY.toFixed(1) +
				"' x2='" + iRight + "' y2='" + fY.toFixed(1) +
				"' stroke='" + COLOUR_AXIS + "' stroke-opacity='0.25' stroke-width='1'/>");
			aOut.push("<text x='" + (iLeft - 8) + "' y='" + (fY + 4).toFixed(1) +
				"' text-anchor='end' font-size='11' fill='" + COLOUR_AXIS + "'>" +
				escapeHtml(fnFormat(fValue)) + "</text>");
		}
		return aOut.join("");
	}

	/**
	 * dateLabels puts a date at each end of the axis and one in the middle. A
	 * 137-day axis cannot carry a label per day.
	 */
	function dateLabels(aPoints, fnX, iBaseline) {
		var iCount = aPoints.length;
		var aOut = [];
		[0, Math.floor(iCount / 2), iCount - 1].forEach(function (i, iIndex) {
			if (i < 0 || i >= iCount) {
				return;
			}
			var sAnchor = iIndex === 0 ? "start" : (iIndex === 2 ? "end" : "middle");
			aOut.push("<text x='" + fnX(i).toFixed(1) + "' y='" + (iBaseline - 12) +
				"' text-anchor='" + sAnchor + "' font-size='11' fill='" + COLOUR_AXIS + "'>" +
				escapeHtml(aPoints[i].date) + "</text>");
		});
		return aOut.join("");
	}

	/**
	 * heatColour maps achievement against target onto the calendar's scale.
	 *
	 * The steps are deliberate rather than a continuous gradient: a planner
	 * reads "that day was under 70 %", not "that day was a slightly deeper
	 * orange than the one beside it".
	 */
	function heatColour(fPct) {
		if (fPct >= 100) {
			return "#256F3A";
		}
		if (fPct >= 95) {
			return "#5EA85E";
		}
		if (fPct >= 85) {
			return "#F0AB00";
		}
		if (fPct >= 70) {
			return "#E76500";
		}
		return "#BB0000";
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
				gridLines(iLeft, iWidth - iRight, y, fMax, formatTons) +
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
				dateLabels(aPoints, x, iHeight) +
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

			return "<div class='sugarChart' role='img' aria-label='Warehouse stock against capacity'>" +
				"<svg viewBox='0 0 " + iWidth + " " + iHeight + "' width='100%' height='" + iHeight + "'>" +
				"<line x1='" + iLeft + "' y1='" + (iTop + iPlotH) + "' x2='" + (iWidth - iRight) +
				"' y2='" + (iTop + iPlotH) + "' stroke='" + COLOUR_AXIS + "' stroke-width='1'/>" +
				sCapacity +
				"<path d='" + aPath.join(" ") + "' fill='none' stroke='" + COLOUR_TARGET + "' stroke-width='2.5'/>" +
				dateLabels(aDays, x, iHeight) +
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
		},

		/**
		 * dailyTrend draws each day's actual as a bar against the day's target as
		 * a line.
		 *
		 * The cumulative curve answers "are we on track for the season"; this one
		 * answers "how did yesterday go", and they are different questions. A
		 * season that is 2 % behind in total can be made of a fortnight at 80 %
		 * and a fortnight at 120 %, and only the daily view shows that.
		 */
		dailyTrend: function (aPoints, aRolling) {
			if (!aPoints || !aPoints.length) {
				return "<div class='sugarChartEmpty'>No plan data for this season yet.</div>";
			}

			var iWidth = 960, iHeight = 240;
			var iLeft = 70, iRight = 20, iTop = 16, iBottom = 34;
			var iPlotW = iWidth - iLeft - iRight;
			var iPlotH = iHeight - iTop - iBottom;

			var fMax = 0;
			aPoints.forEach(function (oPoint) {
				fMax = Math.max(fMax, toNumber(oPoint.target), toNumber(oPoint.actual));
			});
			fMax = fMax * 1.05 || 1;

			var iCount = aPoints.length;
			var fSlot = iPlotW / iCount;
			function x(i) {
				return iLeft + fSlot * (i + 0.5);
			}
			function y(fValue) {
				return iTop + iPlotH - (iPlotH * fValue) / fMax;
			}

			// A 137-day season leaves under 7 px a bar, so they touch. That is
			// the point: the shape of the season reads as a block, and a bad
			// week shows as a notch in it.
			var fBar = Math.max(1, Math.min(18, fSlot - 1));

			var aBars = [];
			var aTargetPath = [];
			aPoints.forEach(function (oPoint, i) {
				aTargetPath.push((i === 0 ? "M" : "L") +
					x(i).toFixed(1) + " " + y(toNumber(oPoint.target)).toFixed(1));
				if (!oPoint.hasActual) {
					return;
				}
				var fActual = toNumber(oPoint.actual);
				var fH = Math.max(0, (fActual / fMax) * iPlotH);
				// Short of the day's target is the thing worth seeing, so it is
				// the thing that changes colour.
				var sColour = fActual < toNumber(oPoint.target) ? COLOUR_SHORT : COLOUR_ACTUAL;
				aBars.push("<rect x='" + (x(i) - fBar / 2).toFixed(1) + "' y='" +
					(iTop + iPlotH - fH).toFixed(1) + "' width='" + fBar.toFixed(1) +
					"' height='" + fH.toFixed(1) + "' fill='" + sColour + "'><title>" +
					escapeHtml(oPoint.date + ": " + formatTons(fActual) +
						" against " + formatTons(toNumber(oPoint.target))) +
					"</title></rect>");
			});

			// The rolling average is the line a forecast is made from, so it is
			// drawn where the forecast can be checked against it.
			var aRollingPath = [];
			if (aRolling && aRolling.length === iCount) {
				aPoints.forEach(function (oPoint, i) {
					if (!oPoint.hasActual) {
						return;
					}
					aRollingPath.push((aRollingPath.length === 0 ? "M" : "L") +
						x(i).toFixed(1) + " " + y(toNumber(aRolling[i])).toFixed(1));
				});
			}

			return "<div class='sugarChart' role='img' aria-label='" +
				escapeHtml("Daily cane crushed against the daily target, " + iCount + " days") + "'>" +
				"<svg viewBox='0 0 " + iWidth + " " + iHeight + "' width='100%' height='" + iHeight + "'>" +
				gridLines(iLeft, iWidth - iRight, y, fMax, formatTons) +
				aBars.join("") +
				"<path d='" + aTargetPath.join(" ") + "' fill='none' stroke='" + COLOUR_TARGET +
				"' stroke-width='2' stroke-dasharray='6 4'/>" +
				(aRollingPath.length
					? "<path d='" + aRollingPath.join(" ") + "' fill='none' stroke='" +
					COLOUR_ROLLING + "' stroke-width='2'/>"
					: "") +
				dateLabels(aPoints, x, iHeight) +
				"</svg></div>";
		},

		/**
		 * recoveryTrend draws the recovery achieved each day against the range it
		 * is meant to stay inside.
		 *
		 * The band matters more than the line. A recovery of 10.4 % means nothing
		 * on its own; a recovery of 10.4 % when the floor is 9.8 % and the target
		 * is 11.0 % means the raw house is losing sugar but not alarmingly.
		 */
		recoveryTrend: function (aPoints, fMinPct, fMaxPct) {
			if (!aPoints || !aPoints.length) {
				return "<div class='sugarChartEmpty'>No recovery recorded yet.</div>";
			}

			var iWidth = 960, iHeight = 240;
			var iLeft = 70, iRight = 20, iTop = 16, iBottom = 34;
			var iPlotW = iWidth - iLeft - iRight;
			var iPlotH = iHeight - iTop - iBottom;

			// The axis is anchored on the range rather than on zero: a recovery
			// chart from 0 to 12 % is a flat line near the top of the plot.
			var fLo = toNumber(fMinPct) || 0;
			var fHi = toNumber(fMaxPct) || 0;
			aPoints.forEach(function (oPoint) {
				if (oPoint.hasActual) {
					fLo = Math.min(fLo, toNumber(oPoint.actual));
					fHi = Math.max(fHi, toNumber(oPoint.actual));
				}
				fLo = Math.min(fLo, toNumber(oPoint.target));
				fHi = Math.max(fHi, toNumber(oPoint.target));
			});
			var fPad = Math.max(0.2, (fHi - fLo) * 0.1);
			fLo -= fPad;
			fHi += fPad;
			if (fHi <= fLo) {
				fHi = fLo + 1;
			}

			var iCount = aPoints.length;
			function x(i) {
				return iLeft + (iCount <= 1 ? 0 : (iPlotW * i) / (iCount - 1));
			}
			function y(fValue) {
				return iTop + iPlotH - (iPlotH * (fValue - fLo)) / (fHi - fLo);
			}

			var sBand = "";
			if (toNumber(fMaxPct) > toNumber(fMinPct)) {
				var fTop = y(toNumber(fMaxPct));
				var fBottom = y(toNumber(fMinPct));
				sBand = "<rect x='" + iLeft + "' y='" + fTop.toFixed(1) +
					"' width='" + iPlotW + "' height='" + (fBottom - fTop).toFixed(1) +
					"' fill='" + COLOUR_ACTUAL + "' fill-opacity='0.08'/>";
			}

			var aTargetPath = [];
			var aActualPath = [];
			var aPoints0 = [];
			aPoints.forEach(function (oPoint, i) {
				aTargetPath.push((i === 0 ? "M" : "L") +
					x(i).toFixed(1) + " " + y(toNumber(oPoint.target)).toFixed(1));
				if (!oPoint.hasActual) {
					return;
				}
				var fValue = toNumber(oPoint.actual);
				aActualPath.push((aActualPath.length === 0 ? "M" : "L") +
					x(i).toFixed(1) + " " + y(fValue).toFixed(1));
				// A day outside the operating range is marked, because that is
				// the day somebody has to explain.
				if (fValue < toNumber(fMinPct) || fValue > toNumber(fMaxPct)) {
					aPoints0.push("<circle cx='" + x(i).toFixed(1) + "' cy='" + y(fValue).toFixed(1) +
						"' r='3.5' fill='" + COLOUR_SHORT + "'><title>" +
						escapeHtml(oPoint.date + ": " + fValue.toFixed(3) + " %") + "</title></circle>");
				}
			});

			return "<div class='sugarChart' role='img' aria-label='Daily raw sugar recovery against its range'>" +
				"<svg viewBox='0 0 " + iWidth + " " + iHeight + "' width='100%' height='" + iHeight + "'>" +
				sBand +
				gridLines(iLeft, iWidth - iRight, y, fHi, function (fValue) {
					return fValue.toFixed(2) + " %";
				}, fLo) +
				"<path d='" + aTargetPath.join(" ") + "' fill='none' stroke='" + COLOUR_TARGET +
				"' stroke-width='2' stroke-dasharray='6 4'/>" +
				(aActualPath.length
					? "<path d='" + aActualPath.join(" ") + "' fill='none' stroke='" +
					COLOUR_ACTUAL + "' stroke-width='2.5'/>"
					: "") +
				aPoints0.join("") +
				dateLabels(aPoints, x, iHeight) +
				"</svg></div>";
		},

		/**
		 * productMix stacks each product's daily output.
		 *
		 * Stacked rather than side by side because the question is what the
		 * factory made in total and how it was divided, not how three products
		 * compare on a Tuesday.
		 */
		productMix: function (aSeries) {
			if (!aSeries || !aSeries.length || !aSeries[0].points || !aSeries[0].points.length) {
				return "<div class='sugarChartEmpty'>No finished goods recorded yet.</div>";
			}

			var iWidth = 960, iHeight = 260;
			var iLeft = 70, iRight = 20, iTop = 16, iBottom = 52;
			var iPlotW = iWidth - iLeft - iRight;
			var iPlotH = iHeight - iTop - iBottom;
			var iCount = aSeries[0].points.length;

			// The tallest day is the sum of every band on that day, not the
			// tallest band: a scale set by one product would clip the stack.
			var fMax = 0;
			var aDayTotals = [];
			for (var d = 0; d < iCount; d++) {
				var fTotal = 0;
				aSeries.forEach(function (oSeries) {
					var oPoint = oSeries.points[d];
					if (oPoint && oPoint.hasActual) {
						fTotal += toNumber(oPoint.actual);
					}
				});
				aDayTotals.push(fTotal);
				fMax = Math.max(fMax, fTotal);
			}
			fMax = fMax * 1.05 || 1;

			var fSlot = iPlotW / iCount;
			var fBar = Math.max(1, Math.min(18, fSlot - 1));
			function x(i) {
				return iLeft + fSlot * (i + 0.5);
			}
			function y(fValue) {
				return iTop + iPlotH - (iPlotH * fValue) / fMax;
			}

			var aBars = [];
			var aRunning = aDayTotals.map(function () { return 0; });
			aSeries.forEach(function (oSeries, iBand) {
				var sColour = BAND_COLOURS[iBand % BAND_COLOURS.length];
				oSeries.points.forEach(function (oPoint, i) {
					if (!oPoint.hasActual) {
						return;
					}
					var fValue = toNumber(oPoint.actual);
					if (fValue <= 0) {
						return;
					}
					var fBase = aRunning[i];
					aRunning[i] = fBase + fValue;
					aBars.push("<rect x='" + (x(i) - fBar / 2).toFixed(1) + "' y='" +
						y(fBase + fValue).toFixed(1) + "' width='" + fBar.toFixed(1) +
						"' height='" + Math.max(0, (fValue / fMax) * iPlotH).toFixed(1) +
						"' fill='" + sColour + "'><title>" +
						escapeHtml(oPoint.date + " " + oSeries.code + ": " + formatTons(fValue)) +
						"</title></rect>");
				});
			});

			var aLegend = aSeries.map(function (oSeries, iBand) {
				var fX = iLeft + iBand * 130;
				return "<rect x='" + fX + "' y='" + (iHeight - 22) +
					"' width='10' height='10' fill='" + BAND_COLOURS[iBand % BAND_COLOURS.length] + "'/>" +
					"<text x='" + (fX + 15) + "' y='" + (iHeight - 13) +
					"' font-size='11' fill='" + COLOUR_AXIS + "'>" +
					escapeHtml(oSeries.code) + "</text>";
			});

			return "<div class='sugarChart' role='img' aria-label='Daily finished goods output by product'>" +
				"<svg viewBox='0 0 " + iWidth + " " + iHeight + "' width='100%' height='" + iHeight + "'>" +
				gridLines(iLeft, iWidth - iRight, y, fMax, formatTons) +
				aBars.join("") +
				dateLabels(aSeries[0].points, x, iHeight - 26) +
				aLegend.join("") +
				"</svg></div>";
		},

		/**
		 * channelTrend draws one cumulative actual line per shipment channel.
		 *
		 * Cumulative rather than daily: shipment is lumpy - a vessel loads over
		 * three days and then nothing for a week - and a daily chart of it is
		 * noise. What the shipment planner needs is whether each channel is
		 * keeping up.
		 */
		channelTrend: function (aSeries) {
			if (!aSeries || !aSeries.length || !aSeries[0].points || !aSeries[0].points.length) {
				return "<div class='sugarChartEmpty'>No shipment planned in this period.</div>";
			}

			var iWidth = 960, iHeight = 260;
			var iLeft = 70, iRight = 20, iTop = 16, iBottom = 52;
			var iPlotW = iWidth - iLeft - iRight;
			var iPlotH = iHeight - iTop - iBottom;
			var iCount = aSeries[0].points.length;

			var fMax = 0;
			aSeries.forEach(function (oSeries) {
				oSeries.points.forEach(function (oPoint) {
					fMax = Math.max(fMax, toNumber(oPoint.cumTarget), toNumber(oPoint.cumActual));
				});
			});
			fMax = fMax * 1.05 || 1;

			function x(i) {
				return iLeft + (iCount <= 1 ? 0 : (iPlotW * i) / (iCount - 1));
			}
			function y(fValue) {
				return iTop + iPlotH - (iPlotH * fValue) / fMax;
			}

			var aPaths = [];
			aSeries.forEach(function (oSeries, iBand) {
				var sColour = BAND_COLOURS[iBand % BAND_COLOURS.length];
				var aPlan = [];
				var aActual = [];
				oSeries.points.forEach(function (oPoint, i) {
					aPlan.push((i === 0 ? "M" : "L") +
						x(i).toFixed(1) + " " + y(toNumber(oPoint.cumTarget)).toFixed(1));
					if (oPoint.hasActual) {
						aActual.push((aActual.length === 0 ? "M" : "L") +
							x(i).toFixed(1) + " " + y(toNumber(oPoint.cumActual)).toFixed(1));
					}
				});
				// The plan is dashed and the actual solid, the same convention as
				// the cane curve above it.
				aPaths.push("<path d='" + aPlan.join(" ") + "' fill='none' stroke='" + sColour +
					"' stroke-width='1.5' stroke-dasharray='6 4' stroke-opacity='0.6'/>");
				if (aActual.length) {
					aPaths.push("<path d='" + aActual.join(" ") + "' fill='none' stroke='" +
						sColour + "' stroke-width='2.5'/>");
				}
			});

			var aLegend = aSeries.map(function (oSeries, iBand) {
				var fX = iLeft + iBand * 150;
				return "<rect x='" + fX + "' y='" + (iHeight - 22) +
					"' width='10' height='10' fill='" + BAND_COLOURS[iBand % BAND_COLOURS.length] + "'/>" +
					"<text x='" + (fX + 15) + "' y='" + (iHeight - 13) +
					"' font-size='11' fill='" + COLOUR_AXIS + "'>" +
					escapeHtml(oSeries.code) + "</text>";
			});

			return "<div class='sugarChart' role='img' aria-label='Cumulative shipment by channel'>" +
				"<svg viewBox='0 0 " + iWidth + " " + iHeight + "' width='100%' height='" + iHeight + "'>" +
				gridLines(iLeft, iWidth - iRight, y, fMax, formatTons) +
				aPaths.join("") +
				dateLabels(aSeries[0].points, x, iHeight - 26) +
				aLegend.join("") +
				"</svg></div>";
		},

		/**
		 * downtimePareto ranks the reasons by the hours they cost, with the
		 * cumulative share drawn over them.
		 *
		 * The cumulative line is the whole point of a Pareto: it says how much of
		 * the problem the first two or three bars account for, which is how a
		 * maintenance meeting decides what to work on.
		 */
		downtimePareto: function (aReasons) {
			if (!aReasons || !aReasons.length) {
				return "<div class='sugarChartEmpty'>No downtime recorded in this period.</div>";
			}

			// Beyond about eight bars a Pareto stops being readable, and the tail
			// is by definition the part that does not matter. It is gathered
			// rather than dropped, so the chart still adds up to the whole.
			var iShown = Math.min(8, aReasons.length);
			var aBarsData = aReasons.slice(0, iShown);
			if (aReasons.length > iShown) {
				var oRest = { reasonCode: "Other", reasonName: "", hours: 0, eventCount: 0 };
				aReasons.slice(iShown).forEach(function (oReason) {
					oRest.hours += toNumber(oReason.hours);
					oRest.eventCount += oReason.eventCount || 0;
				});
				oRest.cumulativeSharePct = aReasons[aReasons.length - 1].cumulativeSharePct;
				aBarsData = aBarsData.concat([oRest]);
			}

			var iWidth = 760, iHeight = 270;
			var iLeft = 70, iRight = 46, iTop = 16, iBottom = 48;
			var iPlotW = iWidth - iLeft - iRight;
			var iPlotH = iHeight - iTop - iBottom;

			var fMax = 0;
			aBarsData.forEach(function (oReason) {
				fMax = Math.max(fMax, toNumber(oReason.hours));
			});
			fMax = fMax * 1.05 || 1;

			var fSlot = iPlotW / aBarsData.length;
			var fBar = Math.min(46, fSlot * 0.6);
			function x(i) {
				return iLeft + fSlot * (i + 0.5);
			}
			function y(fValue) {
				return iTop + iPlotH - (iPlotH * fValue) / fMax;
			}
			// The cumulative line runs on its own 0-100 axis on the right.
			function yShare(fPct) {
				return iTop + iPlotH - (iPlotH * Math.min(100, fPct)) / 100;
			}

			var aBars = [];
			var aLine = [];
			aBarsData.forEach(function (oReason, i) {
				var fHours = toNumber(oReason.hours);
				var fH = (fHours / fMax) * iPlotH;
				aBars.push("<rect x='" + (x(i) - fBar / 2).toFixed(1) + "' y='" +
					(iTop + iPlotH - fH).toFixed(1) + "' width='" + fBar.toFixed(1) +
					"' height='" + Math.max(0, fH).toFixed(1) + "' fill='" + COLOUR_SHORT +
					"' fill-opacity='0.85'><title>" +
					escapeHtml((oReason.reasonName || oReason.reasonCode) + ": " +
						fHours.toFixed(2) + " h over " + (oReason.eventCount || 0) + " event(s)") +
					"</title></rect>");
				aLine.push((i === 0 ? "M" : "L") + x(i).toFixed(1) + " " +
					yShare(toNumber(oReason.cumulativeSharePct)).toFixed(1));
				aBars.push("<text x='" + x(i).toFixed(1) + "' y='" + (iHeight - 30) +
					"' text-anchor='middle' font-size='10' fill='" + COLOUR_AXIS + "'>" +
					escapeHtml(String(oReason.reasonCode).slice(0, 11)) + "</text>");
			});

			// The right-hand axis, labelled so the line is readable as a share.
			var aRight = [];
			[0, 50, 100].forEach(function (iPct) {
				aRight.push("<text x='" + (iWidth - iRight + 6) + "' y='" +
					(yShare(iPct) + 4).toFixed(1) + "' font-size='10' fill='" + COLOUR_ROLLING + "'>" +
					iPct + "%</text>");
			});

			return "<div class='sugarChart' role='img' aria-label='Downtime hours by reason, worst first'>" +
				"<svg viewBox='0 0 " + iWidth + " " + iHeight + "' width='100%' height='" + iHeight + "'>" +
				gridLines(iLeft, iWidth - iRight, y, fMax, function (fValue) {
					return fValue.toFixed(1) + " h";
				}) +
				aBars.join("") +
				"<path d='" + aLine.join(" ") + "' fill='none' stroke='" + COLOUR_ROLLING +
				"' stroke-width='2'/>" +
				aRight.join("") +
				"<text x='" + iLeft + "' y='" + (iHeight - 10) + "' font-size='10' fill='" +
				COLOUR_AXIS + "'>Bars: hours lost. Line: cumulative share of all downtime.</text>" +
				"</svg></div>";
		},

		/**
		 * calendar draws a day-per-cell heat map of achievement against target.
		 *
		 * A 137-day line chart shows the shape of a season; this shows which
		 * particular days went wrong, and a run of red cells down one column is a
		 * weekday problem that no line chart would ever reveal.
		 */
		calendar: function (aPoints) {
			if (!aPoints || !aPoints.length) {
				return "<div class='sugarChartEmpty'>No plan data for this season yet.</div>";
			}

			var iCell = 15, iGap = 2, iTop = 22, iLeft = 34;
			// Columns are weeks and rows are weekdays, so a cell's position
			// carries the day of the week without a label per cell.
			var oFirst = new Date(aPoints[0].date + "T00:00:00Z");
			var iFirstDow = isNaN(oFirst.getTime()) ? 0 : oFirst.getUTCDay();

			var aCells = [];
			var iMaxCol = 0;
			aPoints.forEach(function (oPoint, i) {
				var iOffset = iFirstDow + i;
				var iCol = Math.floor(iOffset / 7);
				var iRow = iOffset % 7;
				iMaxCol = Math.max(iMaxCol, iCol);

				var fTarget = toNumber(oPoint.target);
				var sFill = "#e5e5e5";
				var sLabel = oPoint.date + ": no actual recorded";
				if (oPoint.hasActual && fTarget > 0) {
					var fPct = (toNumber(oPoint.actual) / fTarget) * 100;
					sFill = heatColour(fPct);
					sLabel = oPoint.date + ": " + fPct.toFixed(1) + " % of target (" +
						formatTons(toNumber(oPoint.actual)) + " of " + formatTons(fTarget) + ")";
				} else if (fTarget <= 0) {
					sFill = "#f5f5f5";
					sLabel = oPoint.date + ": no crushing planned";
				}

				aCells.push("<rect x='" + (iLeft + iCol * (iCell + iGap)) + "' y='" +
					(iTop + iRow * (iCell + iGap)) + "' width='" + iCell + "' height='" + iCell +
					"' rx='2' fill='" + sFill + "'><title>" + escapeHtml(sLabel) + "</title></rect>");
			});

			var aRowLabels = ["S", "M", "T", "W", "T", "F", "S"].map(function (sDay, iRow) {
				return "<text x='" + (iLeft - 6) + "' y='" + (iTop + iRow * (iCell + iGap) + 11) +
					"' text-anchor='end' font-size='9' fill='" + COLOUR_AXIS + "'>" + sDay + "</text>";
			});

			// A key, because a colour scale nobody can read is decoration.
			var aKey = [];
			[[60, "under 70 %"], [78, "70-85 %"], [90, "85-95 %"], [97, "95-100 %"],
				[105, "at or above"]].forEach(
				function (aStep, i) {
					var fX = iLeft + i * 96;
					var fY = iTop + 7 * (iCell + iGap) + 12;
					aKey.push("<rect x='" + fX + "' y='" + fY + "' width='10' height='10' rx='2' fill='" +
						heatColour(aStep[0]) + "'/>" +
						"<text x='" + (fX + 15) + "' y='" + (fY + 9) + "' font-size='10' fill='" +
						COLOUR_AXIS + "'>" + escapeHtml(aStep[1]) + "</text>");
				});

			var iWidth = iLeft + (iMaxCol + 1) * (iCell + iGap) + 10;
			var iHeight = iTop + 7 * (iCell + iGap) + 34;
			return "<div class='sugarChart' role='img' aria-label='" +
				escapeHtml("Daily achievement against target, " + aPoints.length + " days") + "'>" +
				"<svg viewBox='0 0 " + Math.max(iWidth, 420) + " " + iHeight +
				"' width='100%' height='" + iHeight + "' preserveAspectRatio='xMinYMin meet'>" +
				aRowLabels.join("") + aCells.join("") + aKey.join("") +
				"</svg></div>";
		}
	};
});
