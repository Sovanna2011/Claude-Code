sap.ui.define([
	"sap/ui/core/mvc/Controller",
	"sap/ui/model/json/JSONModel"
], function (Controller, JSONModel) {
	"use strict";

	return Controller.extend("farm.area.dashboard.controller.App", {
		onInit: function () {
			this.getView().addStyleClass(this.getOwnerComponent().getContentDensityClass());
			this.getView().setModel(new JSONModel({ busy: false }), "appView");
		}
	});
});
