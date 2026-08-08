sap.ui.define([
	"sap/ui/core/Control",
	"sap/ui/core/theming/Parameters"
], function (Control, Parameters) {
	"use strict";

	/**
	 * The interactive map on the dashboard.
	 *
	 * It draws the real block boundaries the service stores in PostGIS — GeoJSON polygons, not
	 * pins — with the farm and zone outlines beneath them, and it is drawn by the application
	 * rather than by Google: the boundaries are the system's own data and the map has to work on
	 * an estate network with no route out. Each block links to Google Maps for the satellite view.
	 *
	 * Selection is two-way. Clicking a block fires "blockSelect", which the controller uses to
	 * select the row in the tree; setting selectedBlockId from the tree highlights the block here.
	 */
	var STATUS_COLOURS = {
		Growing: "sapUiPositiveElement",
		Planted: "sapUiPositiveElement",
		ReadyForHarvest: "sapUiCriticalElement",
		Harvested: "sapUiNeutralElement",
		Prepared: "sapUiInformativeElement",
		Fallow: "sapUiNeutralElement"
	};

	return Control.extend("farm.area.dashboard.control.BlockMap", {
		metadata: {
			properties: {
				blocks: { type: "object", defaultValue: null },     // GeoJSON FeatureCollection
				outlines: { type: "object", defaultValue: null },   // farm and zone boundaries
				selectedBlockId: { type: "int", defaultValue: 0 },
				height: { type: "sap.ui.core.CSSSize", defaultValue: "30rem" }
			},
			events: {
				blockSelect: { parameters: { block: { type: "object" } } }
			}
		},

		renderer: {
			apiVersion: 2,
			render: function (rm, control) {
				rm.openStart("div", control);
				rm.class("faBlockMap");
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
			var features = ((this.getBlocks() || {}).features) || [];
			dom.querySelectorAll("[data-feature-index]").forEach(function (node) {
				var handler = function () {
					var index = parseInt(node.getAttribute("data-feature-index"), 10);
					this.fireBlockSelect({ block: features[index].properties });
				}.bind(this);
				node.addEventListener("click", handler);
				node.addEventListener("keydown", function (e) {
					if (e.key === "Enter" || e.key === " ") {
						e.preventDefault();
						handler();
					}
				});
			}.bind(this));
		},

		_colour: function (name, fallback) {
			return Parameters.get({ name: name }) || fallback;
		},

		/**
		 * Equirectangular, with longitude scaled by the cosine of the centre latitude. Over an
		 * estate that is accurate to well under a metre of relative position, and unlike Web
		 * Mercator the arithmetic stays legible.
		 */
		_svg: function () {
			var collection = this.getBlocks() || { features: [] };
			var features = collection.features || [];
			if (!features.length) {
				return '<div class="faChartEmpty">No block has a boundary or coordinates for the current filter.</div>';
			}

			var rings = [];
			features.forEach(function (feature, index) {
				eachRing(feature.geometry, function (ring) {
					rings.push({ index: index, ring: ring, properties: feature.properties });
				});
			});
			var outlineRings = [];
			eachFeature(this.getOutlines(), function (feature) {
				eachRing(feature.geometry, function (ring) {
					outlineRings.push({ ring: ring, properties: feature.properties });
				});
			});

			var all = rings.concat(outlineRings);
			var bounds = { minLng: Infinity, maxLng: -Infinity, minLat: Infinity, maxLat: -Infinity };
			all.forEach(function (item) {
				item.ring.forEach(function (point) {
					bounds.minLng = Math.min(bounds.minLng, point[0]);
					bounds.maxLng = Math.max(bounds.maxLng, point[0]);
					bounds.minLat = Math.min(bounds.minLat, point[1]);
					bounds.maxLat = Math.max(bounds.maxLat, point[1]);
				});
			});

			var midLat = (bounds.minLat + bounds.maxLat) / 2;
			var kx = Math.cos(midLat * Math.PI / 180);
			var spanX = Math.max((bounds.maxLng - bounds.minLng) * kx, 0.0005);
			var spanY = Math.max(bounds.maxLat - bounds.minLat, 0.0005);
			var pad = 0.06;
			var width = 1000;
			var height = Math.round(width * (spanY * (1 + pad * 2)) / (spanX * (1 + pad * 2)));
			height = Math.min(Math.max(height, 320), 900);

			var project = function (point) {
				var x = ((point[0] * kx) - (bounds.minLng * kx) + spanX * pad) / (spanX * (1 + pad * 2)) * width;
				// Latitude grows northwards and SVG's y grows downwards, so it is inverted here.
				var y = height - ((point[1] - bounds.minLat + spanY * pad) / (spanY * (1 + pad * 2)) * height);
				return x.toFixed(1) + "," + y.toFixed(1);
			};

			var outlineColour = this._colour("sapUiContentForegroundBorderColor", "#8396A8");
			var selectedColour = this._colour("sapUiSelected", "#0064D9");
			var textColour = this._colour("sapUiContentLabelColor", "#556B82");

			var outlineSvg = outlineRings.map(function (item) {
				var isFarm = item.properties.level === "Farm";
				return '<polygon points="' + item.ring.map(project).join(" ") + '" fill="none" stroke="' +
					outlineColour + '" stroke-width="' + (isFarm ? 3 : 1.5) + '" stroke-dasharray="' +
					(isFarm ? "" : "6 4") + '" opacity="' + (isFarm ? 0.9 : 0.6) + '"/>';
			}).join("");

			var selected = this.getSelectedBlockId();
			var blockSvg = rings.map(function (item) {
				var p = item.properties;
				var isSelected = p.blockId === selected;
				var fill = this._colour(STATUS_COLOURS[p.caneStatus] || "sapUiNeutralElement", "#8396A8");
				return '<g data-feature-index="' + item.index + '" class="faMapBlock' + (isSelected ? " faMapBlockOn" : "") +
					'" tabindex="0" role="button" aria-label="' + escapeXml(p.blockCode) + '">' +
					"<title>" + escapeXml(p.blockCode) + " · " + escapeXml(p.blockName) + " · " +
					format(p.totalAreaHa) + " ha · " + escapeXml(p.caneStatus) + "</title>" +
					'<polygon points="' + item.ring.map(project).join(" ") + '" fill="' + fill +
					'" fill-opacity="' + (isSelected ? 0.95 : 0.6) + '" stroke="' +
					(isSelected ? selectedColour : "#FFFFFF") + '" stroke-width="' + (isSelected ? 4 : 1.5) + '"/>' +
					"</g>";
			}.bind(this)).join("");

			// One label per block, at the centroid of its first ring.
			var labels = rings.map(function (item) {
				var centre = centroid(item.ring);
				var xy = project(centre).split(",");
				return '<text x="' + xy[0] + '" y="' + xy[1] + '" text-anchor="middle" class="faMapLabel" fill="' +
					textColour + '" pointer-events="none">' + escapeXml(item.properties.blockCode) + "</text>";
			}).join("");

			var legend = Object.keys({ Growing: 1, Prepared: 1, ReadyForHarvest: 1, Fallow: 1 }).map(function (status) {
				return '<li><i style="background:' + this._colour(STATUS_COLOURS[status], "#8396A8") + '"></i>' +
					escapeXml(status) + "</li>";
			}.bind(this)).join("");

			return '<svg viewBox="0 0 ' + width + " " + height + '" preserveAspectRatio="xMidYMid meet" ' +
				'role="img" aria-label="Map of plantation blocks">' + outlineSvg + blockSvg + labels + "</svg>" +
				'<ul class="faLegend faLegendRow faMapLegend">' + legend + "</ul>";
		}
	});

	function eachFeature(collection, fn) {
		if (collection && collection.features) {
			collection.features.forEach(fn);
		}
	}

	/** Walks Polygon and MultiPolygon alike, handing back each outer ring. */
	function eachRing(geometry, fn) {
		if (!geometry) {
			return;
		}
		if (geometry.type === "Polygon") {
			geometry.coordinates.forEach(function (ring) { fn(ring); });
		} else if (geometry.type === "MultiPolygon") {
			geometry.coordinates.forEach(function (polygon) {
				polygon.forEach(function (ring) { fn(ring); });
			});
		} else if (geometry.type === "Point") {
			// No survey yet: draw a small square so the block is still on the map.
			var d = 0.0004;
			var c = geometry.coordinates;
			fn([[c[0] - d, c[1] - d], [c[0] + d, c[1] - d], [c[0] + d, c[1] + d], [c[0] - d, c[1] + d], [c[0] - d, c[1] - d]]);
		}
	}

	function centroid(ring) {
		var sx = 0, sy = 0;
		ring.forEach(function (p) { sx += p[0]; sy += p[1]; });
		return [sx / ring.length, sy / ring.length];
	}

	function format(value) {
		return (Math.round((value || 0) * 100) / 100).toLocaleString("en-GB");
	}

	function escapeXml(text) {
		return String(text === null || text === undefined ? "" : text).replace(/[&<>"']/g, function (c) {
			return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c];
		});
	}
});
