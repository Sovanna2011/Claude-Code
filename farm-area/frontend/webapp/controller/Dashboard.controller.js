sap.ui.define([
	"sap/ui/core/mvc/Controller",
	"sap/ui/model/json/JSONModel",
	"sap/m/MessageToast",
	"farm/area/dashboard/model/formatter"
], function (Controller, JSONModel, MessageToast, formatter) {
	"use strict";

	// The filter bar, control id → the query parameter it sets. Adding a filter means adding one
	// line here and one Select to the view; nothing else has to change, because every request is
	// built from this map.
	var FILTER_FIELDS = {
		filterCompany: "companyId",
		filterPlantation: "plantationId",
		filterFarm: "farmId",
		filterZone: "zoneId",
		filterBlock: "blockId",
		filterCropYear: "cropYear",
		filterPlantingYear: "plantingYear",
		filterSeason: "seasonId",
		filterVariety: "varietyId",
		filterPlantingType: "plantingType",
		filterLandStatus: "landStatus",
		filterCaneStatus: "caneStatus"
	};

	return Controller.extend("farm.area.dashboard.controller.Dashboard", {
		formatter: formatter,

		onInit: function () {
			var view = this.getView();
			view.addStyleClass(this.getOwnerComponent().getContentDensityClass());

			view.setModel(new JSONModel({ kpis: [], areas: {}, blockCount: 0 }), "kpi");
			view.setModel(new JSONModel({ tree: [] }), "tree");
			view.setModel(new JSONModel({}), "detail");
			view.setModel(new JSONModel({ rows: [], monthly: [], achievementPercent: 0 }), "plan");
			view.setModel(new JSONModel({ filterSummary: "", treeSummary: "", estateMapUrl: "" }), "view");
			view.setModel(new JSONModel({
				companies: [], plantations: [], farms: [], zones: [], blocks: [],
				seasons: [], varieties: [], cropYears: [], plantingYears: [],
				plantingTypes: [
					{ id: "NewPlanting", display: "New planting" },
					{ id: "Ratoon", display: "Ratoon" }
				],
				landStatuses: ["Active", "Reserved", "Retired"].map(toItem),
				caneStatuses: ["Fallow", "Prepared", "Planted", "Growing", "ReadyForHarvest", "Harvested"].map(toItem)
			}), "lookups");

			this.getOwnerComponent().getRouter().getRoute("dashboard")
				.attachPatternMatched(this._onRouteMatched, this);
		},

		_api: function () {
			return this.getOwnerComponent().getApi();
		},

		_onRouteMatched: function () {
			if (!this._api().isSignedIn()) {
				this.getOwnerComponent().getRouter().navTo("login");
				return;
			}
			this.getOwnerComponent().getModel("session").setProperty("/user", this._api().currentUser());

			if (!this._loaded) {
				this._loaded = true;
				this._loadLookups().then(this._reload.bind(this));
			}
		},

		// ---------------------------------------------------------------- loading

		_loadLookups: function () {
			var api = this._api();
			var model = this.getView().getModel("lookups");

			return Promise.all([
				api.get("lookups/companies"),
				api.get("lookups/plantations"),
				api.get("lookups/farms"),
				api.get("lookups/zones"),
				api.get("lookups/blocks"),
				api.get("lookups/seasons"),
				api.get("lookups/varieties"),
				api.get("lookups/cropYears"),
				api.get("lookups/plantingYears")
			]).then(function (results) {
				var keys = ["companies", "plantations", "farms", "zones", "blocks",
					"seasons", "varieties", "cropYears", "plantingYears"];
				keys.forEach(function (key, i) {
					// Every dropdown opens with a blank entry, so a filter can be cleared as
					// easily as it was set.
					model.setProperty("/" + key, [{ id: "", display: "" }].concat(results[i] || []));
				});

				// Default to the most recent crop year rather than every year at once: a
				// plantation's dashboard is about the season being managed.
				var years = model.getProperty("/cropYears");
				if (years.length > 1) {
					this.byId("filterCropYear").setSelectedKey(String(years[1].id));
				}
			}.bind(this)).catch(this._showError.bind(this));
		},

		/** The filter bar's current state, as the query every endpoint receives. */
		_filter: function () {
			var filter = {};
			Object.keys(FILTER_FIELDS).forEach(function (id) {
				var control = this.byId(id);
				var key = control && control.getSelectedKey();
				if (key) {
					filter[FILTER_FIELDS[id]] = key;
				}
			}.bind(this));

			var search = this.byId("treeSearch").getValue().trim();
			if (search) {
				filter.search = search;
			}
			return filter;
		},

		/**
		 * One reload for the whole screen. The KPI cards, the charts, the tree, the map and the
		 * plan-versus-actual section are fetched with the same filter in one pass, so they can
		 * never end up describing different land.
		 */
		_reload: function () {
			var api = this._api();
			var filter = this._filter();
			var view = this.getView();
			var page = this.byId("dashboardPage");

			page.setBusy(true);
			this.byId("errorStrip").setVisible(false);

			return Promise.all([
				api.get("dashboard/farm-area", filter),
				api.get("dashboard/charts", filter),
				api.get("reports/farm-area-tree", filter),
				api.get("dashboard/farm-area/map", filter),
				api.get("reports/planting-plan-vs-actual",
					Object.assign({ level: this.byId("planLevel").getSelectedKey() }, filter))
			]).then(function (results) {
				var kpi = results[0], charts = results[1], tree = results[2], map = results[3], plan = results[4];

				view.getModel("kpi").setData(kpi);
				view.getModel("tree").setData({ tree: tree || [] });
				view.getModel("plan").setData(plan);

				this._applyCharts(charts, plan);

				var blockMap = this.byId("blockMap");
				blockMap.setBlocks(map.blocks);
				blockMap.setOutlines(map.outlines);

				this._describe(kpi, tree || []);
				page.setBusy(false);
			}.bind(this)).catch(function (error) {
				page.setBusy(false);
				this._showError(error);
			}.bind(this));
		},

		_applyCharts: function (charts, plan) {
			var byKey = {};
			(charts || []).forEach(function (series) { byKey[series.key] = series.points; });

			this.byId("chartLandUtilisation").setPoints(byKey.landUtilisation || [])
				.setColours(["sapUiPositiveElement", "sapUiInformativeElement", "sapUiNegativeElement"]);
			this.byId("chartCaneArea").setPoints(byKey.caneArea || [])
				.setColours(["sapUiChartPaletteQualitativeHue1", "sapUiChartPaletteQualitativeHue3"]);
			this.byId("chartAreaByFarm").setPoints(byKey.areaByFarm || []);
			this.byId("chartAreaByZone").setPoints(byKey.areaByZone || []);

			// The monthly chart is the same figures the plan-versus-actual table shows, so the two
			// are read from one response rather than fetched twice.
			var monthly = (plan && plan.monthly) || [];
			this.byId("chartMonthly").setPoints(monthly.map(function (row) {
				return {
					category: row.name ? row.name.slice(0, 3) : String(row.month),
					value: row.actualAreaWithCaneHa,
					extra: row.plannedAreaWithCaneHa
				};
			}));
		},

		_describe: function (kpi, tree) {
			var applied = Object.keys(this._filter()).length;
			var farms = tree.length;
			var zones = tree.reduce(function (sum, farm) { return sum + farm.children.length; }, 0);

			this.getView().getModel("view").setData({
				filterSummary: applied ? applied + " filter(s) applied" : "No filter applied",
				treeSummary: farms + " farm(s) · " + zones + " zone(s) · " + kpi.blockCount +
					" block(s) · " + formatter.hectares(kpi.areas.totalAreaHa) + " ha total",
				estateMapUrl: tree.length && tree[0].mapUrl ? tree[0].mapUrl : ""
			});
		},

		_showError: function (error) {
			var strip = this.byId("errorStrip");
			var message = (error && error.message) || "The request failed.";
			if (error && error.fields && error.fields.length) {
				message += " " + error.fields.map(function (f) { return f.message; }).join(" ");
			}
			strip.setText(message);
			strip.setVisible(true);
		},

		// ---------------------------------------------------------------- filters

		onFilterChanged: function () {
			this._reload();
		},

		// Choosing a farm narrows the zone list, and choosing a zone narrows the blocks: a filter
		// bar that offers zones from other farms invites an empty report.
		onFarmFilterChanged: function () {
			var farmId = this.byId("filterFarm").getSelectedKey();
			this.byId("filterZone").setSelectedKey("");
			this.byId("filterBlock").setSelectedKey("");

			this._api().get("lookups/zones", farmId ? { parentId: farmId } : {}).then(function (zones) {
				this.getView().getModel("lookups").setProperty("/zones", [{ id: "", display: "" }].concat(zones || []));
				return this._api().get("lookups/blocks", {});
			}.bind(this)).then(function (blocks) {
				this.getView().getModel("lookups").setProperty("/blocks", [{ id: "", display: "" }].concat(blocks || []));
				this._reload();
			}.bind(this)).catch(this._showError.bind(this));
		},

		onZoneFilterChanged: function () {
			var zoneId = this.byId("filterZone").getSelectedKey();
			this.byId("filterBlock").setSelectedKey("");

			this._api().get("lookups/blocks", zoneId ? { parentId: zoneId } : {}).then(function (blocks) {
				this.getView().getModel("lookups").setProperty("/blocks", [{ id: "", display: "" }].concat(blocks || []));
				this._reload();
			}.bind(this)).catch(this._showError.bind(this));
		},

		onApplyFilters: function () {
			this._reload();
		},

		onClearFilters: function () {
			Object.keys(FILTER_FIELDS).forEach(function (id) {
				this.byId(id).setSelectedKey("");
			}.bind(this));
			this.byId("treeSearch").setValue("");
			this._reload();
		},

		onSearch: function () {
			this._reload();
		},

		onSearchLive: function (event) {
			// Debounced, so typing a block code does not fire a request per keystroke.
			clearTimeout(this._searchTimer);
			var value = event.getParameter("newValue");
			this._searchTimer = setTimeout(function () {
				if (value.length === 0 || value.length >= 2) {
					this._reload();
				}
			}.bind(this), 350);
		},

		// ---------------------------------------------------------------- drill-down

		onKpiPress: function (event) {
			var kpi = event.getSource().getBindingContext("kpi").getObject();

			// A KPI card is a way into the detail: the cane cards set the matching planting-type
			// filter, and every card scrolls the tree into view.
			if (kpi.key === "newPlanting" || kpi.key === "ratoon") {
				this.byId("filterPlantingType").setSelectedKey(kpi.key === "ratoon" ? "Ratoon" : "NewPlanting");
				this._reload();
			}
			this.byId("treeTable").focus();
			MessageToast.show(kpi.label + ": " + formatter.hectares(kpi.areaHa) + " ha (" +
				kpi.percentOfTotalArea.toFixed(1) + "% of the total area)");
		},

		onChartSelect: function (event) {
			var point = event.getParameter("point");
			MessageToast.show(point.category + ": " + formatter.hectares(point.value) + " ha");
		},

		onFarmChartSelect: function (event) {
			var point = event.getParameter("point");
			if (point && point.id) {
				this.byId("filterFarm").setSelectedKey(String(point.id));
				this.onFarmFilterChanged();
			}
		},

		onZoneChartSelect: function (event) {
			var point = event.getParameter("point");
			if (point && point.id) {
				this.byId("filterZone").setSelectedKey(String(point.id));
				this.onZoneFilterChanged();
			}
		},

		onPlanLevelChanged: function () {
			this._reload();
		},

		// ---------------------------------------------------------------- tree and map

		onExpandAll: function () {
			// Two levels is the whole hierarchy: farms open to zones, zones to blocks.
			this.byId("treeTable").expandToLevel(2);
		},

		onCollapseAll: function () {
			this.byId("treeTable").collapseAll();
		},

		/** Tree → map: selecting a block row highlights it on the map and fills the detail panel. */
		onTreeRowSelected: function (event) {
			var index = event.getParameter("rowIndex");
			if (index === undefined || index < 0) {
				return;
			}
			var context = this.byId("treeTable").getContextByIndex(index);
			var node = context && context.getObject();
			if (!node) {
				return;
			}
			if (node.nodeType !== "Block") {
				// A farm or zone row still moves the map, by clearing the block selection.
				this.byId("blockMap").setSelectedBlockId(0);
				this.getView().getModel("detail").setData({});
				return;
			}
			this.byId("blockMap").setSelectedBlockId(node.id);
			this._showBlockDetail(node.id);
		},

		/** Map → tree: clicking a block selects its row, expanding the branch to reach it. */
		onMapBlockSelect: function (event) {
			var block = event.getParameter("block");
			this.byId("blockMap").setSelectedBlockId(block.blockId);
			this.getView().getModel("detail").setData(block);
			this._selectTreeRow(block.blockId);
		},

		_selectTreeRow: function (blockId) {
			var table = this.byId("treeTable");
			table.expandToLevel(2);

			// The binding's row count only includes expanded nodes, so the search happens after
			// the expansion above.
			setTimeout(function () {
				var binding = table.getBinding("rows");
				var count = binding ? binding.getLength() : 0;
				for (var i = 0; i < count; i++) {
					var context = table.getContextByIndex(i);
					var node = context && context.getObject();
					if (node && node.nodeType === "Block" && node.id === blockId) {
						table.setSelectedIndex(i);
						table.setFirstVisibleRow(Math.max(0, i - 3));
						return;
					}
				}
			}, 0);
		},

		_showBlockDetail: function (blockId) {
			var features = ((this.byId("blockMap").getBlocks() || {}).features) || [];
			for (var i = 0; i < features.length; i++) {
				if (features[i].properties.blockId === blockId) {
					this.getView().getModel("detail").setData(features[i].properties);
					return;
				}
			}
			// Not on the map — no boundary and no coordinates — so ask the service for it.
			this._api().get("blocks/" + blockId).then(function (block) {
				this.getView().getModel("detail").setData({
					blockId: block.id, blockCode: block.code, blockName: block.name,
					farmName: block.farmName, zoneName: block.zoneName,
					caneStatus: block.caneStatus, mapUrl: block.mapUrl,
					totalAreaHa: block.areas.totalAreaHa,
					newPlantingAreaHa: block.areas.newPlantingAreaHa,
					ratoonAreaHa: block.areas.ratoonAreaHa,
					areaWithCaneHa: block.areas.areaWithCaneHa,
					availableAreaHa: block.areas.availableAreaHa,
					nonPlantableAreaHa: block.areas.nonPlantableAreaHa
				});
			}.bind(this)).catch(this._showError.bind(this));
		},

		// ---------------------------------------------------------------- export

		onExportExcel: function () {
			var button = this.byId("dashboardPage");
			button.setBusy(true);
			this._api().download("reports/farm-area-tree.xlsx", this._filter(), "farm-area-tree.xlsx")
				.then(function (name) {
					button.setBusy(false);
					MessageToast.show("Exported " + name);
				})
				.catch(function (error) {
					button.setBusy(false);
					this._showError(error);
				}.bind(this));
		},

		onSignOut: function () {
			this._api().signOut();
			this._loaded = false;
			this.getOwnerComponent().getRouter().navTo("login");
		}
	});

	function toItem(value) {
		return { id: value, display: value.replace(/([a-z])([A-Z])/g, "$1 $2") };
	}
});
