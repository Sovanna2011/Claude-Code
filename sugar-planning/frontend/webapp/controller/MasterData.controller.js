sap.ui.define([
	"sugarplan/controller/BaseController",
	"sap/m/Column",
	"sap/m/Text",
	"sap/m/ColumnListItem",
	"sap/m/ObjectStatus"
], function (BaseController, Column, Text, ColumnListItem, ObjectStatus) {
	"use strict";

	/**
	 * MasterData is a generic maintenance screen. Because every master entity
	 * is exposed through the same API shape, one screen serves all of them and
	 * a new entity needs no new page.
	 */
	var ENTITIES = [
		{ key: "companies", text: "Companies", columns: ["code", "name", "currency", "timeZone"] },
		{ key: "factories", text: "Factories", columns: ["code", "name", "timeZone"] },
		{ key: "production-lines", text: "Production lines", columns: ["code", "name", "stage", "ratedTph"] },
		{ key: "shifts", text: "Shifts", columns: ["code", "name", "startTime", "hours"] },
		{ key: "product-categories", text: "Product categories", columns: ["code", "name"] },
		{ key: "products", text: "Products", columns: ["code", "name", "categoryCode", "baseUom", "storageClass"] },
		{ key: "units", text: "Units of measure", columns: ["code", "name", "dimension", "decimals"] },
		{ key: "unit-conversions", text: "Unit conversions", columns: ["fromUom", "toUom", "factor"] },
		{ key: "packaging-types", text: "Packaging types", columns: ["code", "name", "netWeightKg"] },
		{ key: "warehouses", text: "Warehouses and silos", columns: ["code", "name", "storageClass", "capacityTons", "usablePct"] },
		{ key: "customers", text: "Customers", columns: ["code", "name", "country"] },
		{ key: "shipment-channels", text: "Shipment channels", columns: ["code", "name", "category"] },
		{ key: "materials", text: "Materials", columns: ["code", "name", "uom", "safetyStock", "leadTimeDays", "onHand"] },
		{ key: "cane-sources", text: "Cane sources", columns: ["code", "name", "sourceType", "zone", "hectares", "expectedYieldTonsPerHectare", "truckCapacityTons", "trucksPerDay"] },
		{ key: "reason-codes", text: "Reason codes", columns: ["code", "name", "category"] }
	];

	return BaseController.extend("sugarplan.controller.MasterData", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({
				entities: ENTITIES,
				selectedEntity: "products",
				search: "",
				items: [],
				count: 0
			}), "view");
			this.getRouter().getRoute("masterdata").attachPatternMatched(this._onDisplay, this);
		},

		_onDisplay: function () {
			this._load();
		},

		_load: function () {
			var oModel = this.getView().getModel("view");
			var sEntity = oModel.getProperty("/selectedEntity");
			var that = this;

			this.setBusy(true);
			this.getService().listMaster(sEntity, { $search: oModel.getProperty("/search") })
				.then(function (oPage) {
					oModel.setProperty("/items", oPage.value || []);
					oModel.setProperty("/count", oPage.count || 0);
					that._buildColumns(sEntity);
					that.setBusy(false);
				})
				.catch(function (oProblem) {
					that.showError(oProblem);
				});
		},

		/** _buildColumns rebuilds the table for the selected entity. */
		_buildColumns: function (sEntity) {
			var oTable = this.byId("masterTable");
			if (!oTable) {
				return;
			}
			var oDefinition = ENTITIES.filter(function (oEntity) {
				return oEntity.key === sEntity;
			})[0];
			if (!oDefinition) {
				return;
			}

			oTable.destroyColumns();
			oTable.unbindItems();

			var aCells = [];
			oDefinition.columns.forEach(function (sField) {
				oTable.addColumn(new Column({ header: new Text({ text: sField }) }));
				aCells.push(new Text({ text: "{view>" + sField + "}" }));
			});
			// Every master record carries effective dating and an active flag.
			oTable.addColumn(new Column({ header: new Text({ text: "Active" }) }));
			aCells.push(new ObjectStatus({
				text: "{= ${view>active} ? 'Active' : 'Inactive' }",
				state: "{= ${view>active} ? 'Success' : 'None' }"
			}));

			oTable.bindItems({
				path: "view>/items",
				templateShareable: false,
				template: new ColumnListItem({ cells: aCells })
			});
		},

		onEntityChange: function () {
			this._load();
		},

		onSearch: function (oEvent) {
			this.getView().getModel("view").setProperty("/search", oEvent.getParameter("newValue") || "");
			if (this._iSearchTimer) {
				clearTimeout(this._iSearchTimer);
			}
			this._iSearchTimer = setTimeout(this._load.bind(this), 300);
		},

		onNavBack: function () {
			this.navTo("launchpad");
		}
	});
});
