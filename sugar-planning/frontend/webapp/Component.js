sap.ui.define([
	"sap/ui/core/UIComponent",
	"sap/ui/Device",
	"sugarplan/model/SugarService"
], function (UIComponent, Device, SugarService) {
	"use strict";

	return UIComponent.extend("sugarplan.Component", {

		metadata: { manifest: "json" },

		init: function () {
			UIComponent.prototype.init.apply(this, arguments);

			// One service instance for the whole application.
			this._oService = new SugarService("/api/v1");

			// A device model drives the responsive behaviour of the shell.
			this.setModel(this._oService.newModel({
				isPhone: Device.system.phone,
				isTablet: Device.system.tablet,
				isDesktop: Device.system.desktop
			}), "device");

			// The app model holds the shell state: who is signed in, which
			// season is selected, and whether a request is in flight.
			this.setModel(this._oService.newModel({
				busy: false,
				signedIn: false,
				profile: null,
				seasons: [],
				selectedSeasonId: "",
				selectedVersionId: "",
				permissions: []
			}), "app");

			this.getRouter().initialize();
		},

		/** getService exposes the API client to the controllers. */
		getService: function () {
			return this._oService;
		},

		/** getContentDensityClass keeps tables compact on a desktop and
		 * touch-friendly on a tablet. */
		getContentDensityClass: function () {
			if (this._sContentDensityClass === undefined) {
				this._sContentDensityClass = Device.support.touch ? "sapUiSizeCozy" : "sapUiSizeCompact";
			}
			return this._sContentDensityClass;
		},

		destroy: function () {
			if (this._oService) {
				this._oService.destroy();
			}
			UIComponent.prototype.destroy.apply(this, arguments);
		}
	});
});
