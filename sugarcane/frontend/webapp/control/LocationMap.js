sap.ui.define([
	"sap/ui/core/Control",
	"sap/ui/core/theming/Parameters"
], function (Control, Parameters) {
	"use strict";

	/**
	 * The interactive map on the dashboard.
	 *
	 * It draws the real boundaries the service stores in PostGIS — GeoJSON polygons, not pins — at
	 * whichever location level was asked for: one shape per farm, per zone, or per block. It is
	 * drawn by the application rather than by Google, because the boundaries are the system's own
	 * data and the map has to work on an estate network with no route out. Each location links to
	 * Google Maps for the satellite view.
	 *
	 * Selection is two-way. Clicking a shape fires "featureSelect", which the controller uses to
	 * narrow the dashboard to that location; setting selectedId from the tree highlights it here.
	 *
	 * At block level the fill says what is growing, because that is the question asked of a block.
	 * At farm and zone level the fill is the share of the location already under cane, because a
	 * whole farm has no single status — it is part planted, and how much is the whole point.
	 */
	var STATUS_COLOURS = {
		Growing: "sapUiPositiveElement",
		Planted: "sapUiPositiveElement",
		ReadyForHarvest: "sapUiCriticalElement",
		Harvested: "sapUiNeutralElement",
		Prepared: "sapUiInformativeElement",
		Fallow: "sapUiNeutralElement"
	};

	// The planted-share ramp, darkest last. Four bands rather than a continuous scale: a reader can
	// tell four fills apart on a small screen, and cannot tell 55% from 62% by colour anyway.
	var SHARE_BANDS = [
		{ upTo: 1, label: "Nothing planted", colour: "sapUiNeutralElement", opacity: 0.35 },
		{ upTo: 40, label: "Under 40% planted", colour: "sapUiPositiveElement", opacity: 0.35 },
		{ upTo: 80, label: "40 to 80% planted", colour: "sapUiPositiveElement", opacity: 0.6 },
		{ upTo: Infinity, label: "Over 80% planted", colour: "sapUiPositiveElement", opacity: 0.9 }
	];

	return Control.extend("farm.area.dashboard.control.LocationMap", {
		metadata: {
			properties: {
				features: { type: "object", defaultValue: null },   // GeoJSON FeatureCollection
				outlines: { type: "object", defaultValue: null },   // the levels above the drawn one
				level: { type: "string", defaultValue: "Block" },   // Farm, Zone or Block
				selectedId: { type: "int", defaultValue: 0 },
				height: { type: "sap.ui.core.CSSSize", defaultValue: "30rem" }
			},
			events: {
				featureSelect: { parameters: { feature: { type: "object" } } }
			}
		},

		renderer: {
			apiVersion: 2,
			render: function (rm, control) {
				rm.openStart("div", control);
				rm.class("faLocationMap");
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
			var features = ((this.getFeatures() || {}).features) || [];
			dom.querySelectorAll("[data-feature-index]").forEach(function (node) {
				var handler = function () {
					var index = parseInt(node.getAttribute("data-feature-index"), 10);
					this.fireFeatureSelect({ feature: features[index].properties });
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

		/** The band a location falls in, by the share of it already under cane. */
		_band: function (properties) {
			var share = properties.plantedPercent || 0;
			for (var i = 0; i < SHARE_BANDS.length; i++) {
				if (share < SHARE_BANDS[i].upTo) {
					return SHARE_BANDS[i];
				}
			}
			return SHARE_BANDS[SHARE_BANDS.length - 1];
		},

		/** How one feature is filled, which depends on the level being drawn. */
		_fillOf: function (properties, isSelected) {
			if (this.getLevel() === "Block") {
				return {
					colour: this._colour(STATUS_COLOURS[properties.caneStatus] || "sapUiNeutralElement", "#8396A8"),
					opacity: isSelected ? 0.95 : 0.6
				};
			}
			var band = this._band(properties);
			return {
				colour: this._colour(band.colour, "#8396A8"),
				opacity: isSelected ? Math.min(band.opacity + 0.25, 1) : band.opacity
			};
		},

		/** The tooltip, which is the only place the figures appear without clicking. */
		_titleOf: function (properties) {
			var parts = [properties.code, properties.name, format(properties.totalAreaHa) + " ha"];
			if (this.getLevel() === "Block") {
				parts.push(properties.caneStatus);
			} else {
				parts.push(properties.blockCount + " block(s)");
				parts.push(format(properties.plantedPercent) + "% planted");
			}
			return parts.join(" · ");
		},

		/**
		 * Equirectangular, with longitude scaled by the cosine of the centre latitude. Over an
		 * estate that is accurate to well under a metre of relative position, and unlike Web
		 * Mercator the arithmetic stays legible.
		 */
		_svg: function () {
			var collection = this.getFeatures() || { features: [] };
			var features = collection.features || [];
			if (!features.length) {
				return '<div class="faChartEmpty">No ' + this.getLevel().toLowerCase() +
					" has a boundary or coordinates for the current filter.</div>";
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

			var selected = this.getSelectedId();
			var shapeSvg = rings.map(function (item) {
				var p = item.properties;
				var isSelected = p.id === selected;
				var fill = this._fillOf(p, isSelected);
				return '<g data-feature-index="' + item.index + '" class="faMapShape' + (isSelected ? " faMapShapeOn" : "") +
					'" tabindex="0" role="button" aria-label="' + escapeXml(p.code) + '">' +
					"<title>" + escapeXml(this._titleOf(p)) + "</title>" +
					'<polygon points="' + item.ring.map(project).join(" ") + '" fill="' + fill.colour +
					'" fill-opacity="' + fill.opacity + '" stroke="' +
					(isSelected ? selectedColour : "#FFFFFF") + '" stroke-width="' + (isSelected ? 4 : 1.5) + '"/>' +
					"</g>";
			}.bind(this)).join("");

			// One label per location, not per ring. A farm drawn as the union of its blocks is a
			// MultiPolygon of a dozen pieces, and labelling every piece writes the same code across
			// the map a dozen times; the largest piece is the one with room for the text.
			var largest = {};
			rings.forEach(function (item) {
				var area = ringArea(item.ring);
				if (!largest[item.index] || area > largest[item.index].area) {
					largest[item.index] = { area: area, ring: item.ring, properties: item.properties };
				}
			});
			var labels = Object.keys(largest).map(function (index) {
				var item = largest[index];
				var xy = project(centroid(item.ring)).split(",");
				return '<text x="' + xy[0] + '" y="' + xy[1] + '" text-anchor="middle" class="faMapLabel" fill="' +
					textColour + '" pointer-events="none">' + escapeXml(item.properties.code) + "</text>";
			}).join("");

			return '<svg viewBox="0 0 ' + width + " " + height + '" preserveAspectRatio="xMidYMid meet" ' +
				'role="img" aria-label="Map of the plantation by ' + this.getLevel().toLowerCase() + '">' +
				outlineSvg + shapeSvg + labels + "</svg>" +
				'<ul class="faLegend faLegendRow faMapLegend">' + this._legend() + "</ul>";
		},

		/** The legend explains whichever fill the current level is using. */
		_legend: function () {
			var entries = this.getLevel() === "Block"
				? ["Growing", "Prepared", "ReadyForHarvest", "Fallow"].map(function (status) {
					return { label: status, colour: this._colour(STATUS_COLOURS[status], "#8396A8"), opacity: 0.6 };
				}.bind(this))
				: SHARE_BANDS.map(function (band) {
					return { label: band.label, colour: this._colour(band.colour, "#8396A8"), opacity: band.opacity };
				}.bind(this));

			return entries.map(function (entry) {
				return '<li><i style="background:' + entry.colour + ";opacity:" + entry.opacity + '"></i>' +
					escapeXml(entry.label) + "</li>";
			}).join("");
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
			// No survey yet: draw a small square so the location is still on the map.
			var d = 0.0004;
			var c = geometry.coordinates;
			fn([[c[0] - d, c[1] - d], [c[0] + d, c[1] - d], [c[0] + d, c[1] + d], [c[0] - d, c[1] + d], [c[0] - d, c[1] - d]]);
		}
	}

	/**
	 * The shoelace area of a ring, in degrees squared. It is only ever compared against another
	 * ring of the same location, so the unit does not matter and no projection is needed.
	 */
	function ringArea(ring) {
		var sum = 0;
		for (var i = 0, j = ring.length - 1; i < ring.length; j = i++) {
			sum += (ring[j][0] * ring[i][1]) - (ring[i][0] * ring[j][1]);
		}
		return Math.abs(sum) / 2;
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
