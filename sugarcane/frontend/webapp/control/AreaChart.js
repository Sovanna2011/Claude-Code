sap.ui.define([
	"sap/ui/core/Control",
	"sap/ui/core/theming/Parameters"
], function (Control, Parameters) {
	"use strict";

	/**
	 * A small analytical chart drawn as inline SVG.
	 *
	 * OpenUI5 does not ship sap.viz — VizFrame is part of the SAP-delivered SAPUI5 distribution —
	 * so the dashboard draws its own. The control takes the same {category, value, extra} points
	 * the API returns, uses the theme's own colour parameters so it matches Horizon in light and
	 * dark, and fires "select" when a segment is clicked, which is what drives the drill-down.
	 */
	var PALETTE = [
		"sapUiChartPaletteQualitativeHue1", "sapUiChartPaletteQualitativeHue2",
		"sapUiChartPaletteQualitativeHue3", "sapUiChartPaletteQualitativeHue4",
		"sapUiChartPaletteQualitativeHue5", "sapUiChartPaletteQualitativeHue6",
		"sapUiChartPaletteQualitativeHue7", "sapUiChartPaletteQualitativeHue8",
		"sapUiChartPaletteQualitativeHue9", "sapUiChartPaletteQualitativeHue10",
		"sapUiChartPaletteQualitativeHue11"
	];

	return Control.extend("farm.area.dashboard.control.AreaChart", {
		metadata: {
			properties: {
				chartType: { type: "string", defaultValue: "donut" }, // donut | bar | column
				title: { type: "string", defaultValue: "" },
				unit: { type: "string", defaultValue: "ha" },
				points: { type: "object", defaultValue: [] },
				extraLabel: { type: "string", defaultValue: "" },
				height: { type: "sap.ui.core.CSSSize", defaultValue: "16rem" },
				colours: { type: "object", defaultValue: null }
			},
			events: {
				select: { parameters: { point: { type: "object" }, index: { type: "int" } } }
			}
		},

		renderer: {
			apiVersion: 2,
			render: function (rm, control) {
				rm.openStart("div", control);
				rm.class("faAreaChart");
				rm.style("height", control.getHeight());
				rm.openEnd();
				rm.unsafeHtml(control._svg());
				rm.close("div");
			}
		},

		onAfterRendering: function () {
			var dom = this.getDomRef();
			if (!dom) {
				return;
			}
			var points = this.getPoints() || [];
			dom.querySelectorAll("[data-point-index]").forEach(function (node) {
				node.addEventListener("click", function () {
					var index = parseInt(node.getAttribute("data-point-index"), 10);
					this.fireSelect({ point: points[index], index: index });
				}.bind(this));
			}.bind(this));
		},

		/** Theme colours, read once per render so the chart follows a theme switch. */
		_colour: function (index) {
			var explicit = this.getColours();
			if (explicit && explicit[index]) {
				return Parameters.get({ name: explicit[index] }) || explicit[index];
			}
			return Parameters.get({ name: PALETTE[index % PALETTE.length] }) || "#5899DA";
		},

		_label: function () {
			return Parameters.get({ name: "sapUiContentLabelColor" }) || "#556B82";
		},

		_svg: function () {
			var points = this.getPoints() || [];
			if (!points.length) {
				return '<div class="faChartEmpty">No data for the current filter.</div>';
			}
			switch (this.getChartType()) {
				case "bar":
					return this._bars(points, true);
				case "column":
					return this._bars(points, false);
				default:
					return this._donut(points);
			}
		},

		_donut: function (points) {
			var total = points.reduce(function (sum, p) { return sum + Math.max(p.value, 0); }, 0);
			var size = 240, radius = 100, inner = 62, cx = size / 2, cy = size / 2;
			var angle = -Math.PI / 2;
			var arcs = "";

			points.forEach(function (p, i) {
				var value = Math.max(p.value, 0);
				if (value <= 0 || total <= 0) {
					return;
				}
				var sweep = (value / total) * Math.PI * 2;
				// A single slice covering the whole circle cannot be drawn as an arc — its start
				// and end points coincide — so it is drawn as a pair of rings instead.
				if (sweep >= Math.PI * 2 - 0.0001) {
					arcs += '<g data-point-index="' + i + '" class="faSlice">' +
						'<circle cx="' + cx + '" cy="' + cy + '" r="' + radius + '" fill="' + this._colour(i) + '"/>' +
						'<circle cx="' + cx + '" cy="' + cy + '" r="' + inner + '" fill="var(--faChartBg, #fff)"/></g>';
					angle += sweep;
					return;
				}
				var end = angle + sweep;
				var large = sweep > Math.PI ? 1 : 0;
				var path = [
					"M", cx + radius * Math.cos(angle), cy + radius * Math.sin(angle),
					"A", radius, radius, 0, large, 1, cx + radius * Math.cos(end), cy + radius * Math.sin(end),
					"L", cx + inner * Math.cos(end), cy + inner * Math.sin(end),
					"A", inner, inner, 0, large, 0, cx + inner * Math.cos(angle), cy + inner * Math.sin(angle),
					"Z"
				].join(" ");
				arcs += '<path d="' + path + '" fill="' + this._colour(i) + '" data-point-index="' + i +
					'" class="faSlice"><title>' + escapeXml(p.category) + ": " + format(p.value) + " " +
					escapeXml(this.getUnit()) + " (" + (total ? Math.round(value / total * 1000) / 10 : 0) +
					"%)</title></path>";
				angle = end;
			}.bind(this));

			var legend = points.map(function (p, i) {
				var share = total ? Math.round(p.value / total * 1000) / 10 : 0;
				return '<li><i style="background:' + this._colour(i) + '"></i>' +
					'<span class="faLegendName">' + escapeXml(p.category) + "</span>" +
					'<span class="faLegendValue">' + format(p.value) + " " + escapeXml(this.getUnit()) +
					" · " + share + "%</span></li>";
			}.bind(this)).join("");

			return '<div class="faDonutLayout">' +
				'<svg viewBox="0 0 ' + size + " " + size + '" role="img" aria-label="' +
				escapeXml(this.getTitle()) + '" preserveAspectRatio="xMidYMid meet">' + arcs +
				'<text x="' + cx + '" y="' + (cy - 4) + '" text-anchor="middle" class="faDonutTotal">' +
				format(total) + "</text>" +
				'<text x="' + cx + '" y="' + (cy + 16) + '" text-anchor="middle" class="faDonutUnit">' +
				escapeXml(this.getUnit()) + "</text></svg>" +
				'<ul class="faLegend">' + legend + "</ul></div>";
		},

		/**
		 * Bars, horizontal or vertical. When a point carries "extra" the two are drawn side by
		 * side — that is how planned sits beside actual on the monthly progress chart.
		 */
		_bars: function (points, horizontal) {
			var hasExtra = points.some(function (p) { return typeof p.extra === "number"; });
			var max = points.reduce(function (m, p) {
				return Math.max(m, p.value, typeof p.extra === "number" ? p.extra : 0);
			}, 0) || 1;

			var width = 620;
			var rowHeight = horizontal ? 28 : 0;
			var height = horizontal ? Math.max(160, points.length * rowHeight + 30) : 260;
			var labelWidth = horizontal ? 150 : 0;
			var plotWidth = width - labelWidth - 90;
			var baseline = height - 46;

			var series = "";
			points.forEach(function (p, i) {
				var main = this._colour(hasExtra ? 0 : i);
				var second = this._colour(1);
				if (horizontal) {
					var y = 12 + i * rowHeight;
					var barHeight = hasExtra ? 9 : 18;
					series += '<g data-point-index="' + i + '" class="faBar">' +
						'<title>' + escapeXml(p.category) + ": " + format(p.value) + " " + escapeXml(this.getUnit()) + "</title>" +
						'<text x="' + (labelWidth - 8) + '" y="' + (y + 14) + '" text-anchor="end" class="faAxisLabel">' +
						escapeXml(clip(p.category, 22)) + "</text>" +
						'<rect x="' + labelWidth + '" y="' + y + '" width="' + (p.value / max * plotWidth) +
						'" height="' + barHeight + '" rx="2" fill="' + main + '"/>';
					if (hasExtra) {
						series += '<rect x="' + labelWidth + '" y="' + (y + barHeight + 2) + '" width="' +
							(p.extra / max * plotWidth) + '" height="' + barHeight + '" rx="2" fill="' + second +
							'" opacity="0.55"/>';
					}
					series += '<text x="' + (labelWidth + plotWidth + 8) + '" y="' + (y + 14) +
						'" class="faAxisValue">' + format(p.value) + "</text></g>";
				} else {
					var slot = plotWidth / points.length;
					var barWidth = hasExtra ? slot * 0.32 : slot * 0.55;
					var x = 54 + i * slot + (slot - (hasExtra ? barWidth * 2 + 4 : barWidth)) / 2;
					var h = (p.value / max) * (baseline - 24);
					series += '<g data-point-index="' + i + '" class="faBar">' +
						'<title>' + escapeXml(p.category) + ": " + format(p.value) + " " + escapeXml(this.getUnit()) + "</title>";
					if (hasExtra) {
						var eh = (p.extra / max) * (baseline - 24);
						series += '<rect x="' + x + '" y="' + (baseline - eh) + '" width="' + barWidth +
							'" height="' + eh + '" rx="2" fill="' + this._colour(1) + '" opacity="0.55"/>' +
							'<rect x="' + (x + barWidth + 4) + '" y="' + (baseline - h) + '" width="' + barWidth +
							'" height="' + h + '" rx="2" fill="' + this._colour(0) + '"/>';
					} else {
						series += '<rect x="' + x + '" y="' + (baseline - h) + '" width="' + barWidth +
							'" height="' + h + '" rx="2" fill="' + main + '"/>';
					}
					series += '<text x="' + (x + (hasExtra ? barWidth : barWidth / 2)) + '" y="' + (baseline + 16) +
						'" text-anchor="middle" class="faAxisLabel">' + escapeXml(clip(p.category, 9)) + "</text></g>";
				}
			}.bind(this));

			var axis = horizontal
				? ""
				: '<line x1="50" y1="' + baseline + '" x2="' + (width - 20) + '" y2="' + baseline +
				  '" class="faAxis"/><text x="44" y="28" text-anchor="end" class="faAxisValue">' + format(max) +
				  '</text><text x="44" y="' + (baseline + 4) + '" text-anchor="end" class="faAxisValue">0</text>';

			var key = hasExtra
				? '<ul class="faLegend faLegendRow"><li><i style="background:' + this._colour(0) + '"></i>Actual</li>' +
				  '<li><i style="background:' + this._colour(1) + ';opacity:.55"></i>' +
				  escapeXml(this.getExtraLabel() || "Planned") + "</li></ul>"
				: "";

			return '<svg viewBox="0 0 ' + width + " " + height + '" role="img" aria-label="' +
				escapeXml(this.getTitle()) + '" preserveAspectRatio="xMidYMid meet">' + axis + series + "</svg>" + key;
		}
	});

	function format(value) {
		return (Math.round(value * 100) / 100).toLocaleString("en-GB", { minimumFractionDigits: 0, maximumFractionDigits: 2 });
	}

	function clip(text, max) {
		text = String(text || "");
		return text.length > max ? text.slice(0, max - 1) + "…" : text;
	}

	function escapeXml(text) {
		return String(text === null || text === undefined ? "" : text).replace(/[&<>"']/g, function (c) {
			return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c];
		});
	}
});
