sap.ui.define([
	"sap/ui/core/UIComponent",
	"sap/ui/Device",
	"sap/ui/model/json/JSONModel",
	"farm/area/dashboard/service/Api"
], function (UIComponent, Device, JSONModel, Api) {
	"use strict";

	return UIComponent.extend("farm.area.dashboard.Component", {
		metadata: { manifest: "json" },

		init: function () {
			UIComponent.prototype.init.apply(this, arguments);

			// One API client for the whole application, so the bearer token is held in a single
			// place and every screen sends it.
			this.api = new Api(this.getManifestEntry("/sap.app/dataSources/farmArea/uri"));

			this.setModel(new JSONModel({
				isPhone: Device.system.phone,
				isTablet: Device.system.tablet,
				isDesktop: Device.system.desktop
			}), "device");

			this.setModel(new JSONModel({ user: null, signedIn: false }), "session");

			this.getRouter().initialize();
		},

		getApi: function () {
			return this.api;
		},

		getContentDensityClass: function () {
			return Device.support.touch ? "sapUiSizeCozy" : "sapUiSizeCompact";
		}
	});
});
