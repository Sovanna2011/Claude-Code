sap.ui.define([
	"sap/ui/core/mvc/Controller",
	"sap/ui/model/json/JSONModel"
], function (Controller, JSONModel) {
	"use strict";

	return Controller.extend("farm.area.dashboard.controller.App", {
		onInit: function () {
			this.getView().addStyleClass(this.getOwnerComponent().getContentDensityClass());
			this.getView().setModel(new JSONModel({ busy: false, layout: "OneColumn" }), "appView");

			// Each route declares the layout it wants; the shell simply follows. Keeping it in one
			// place means a new route sets its own column count without touching any controller.
			var router = this.getOwnerComponent().getRouter();
			router.attachRouteMatched(function (event) {
				var layout = event.getParameter("config").layout;
				if (layout) {
					this.getView().getModel("appView").setProperty("/layout", layout);
				}
			}, this);
		},

		// Dragging the column separator is the user overriding the route's choice; honour it rather
		// than snapping back on the next navigation within the same route.
		onStateChange: function (event) {
			if (event.getParameter("isNavigationArrow")) {
				this.getView().getModel("appView")
					.setProperty("/layout", event.getParameter("layout"));
			}
		}
	});
});
