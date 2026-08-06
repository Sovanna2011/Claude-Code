sap.ui.define([
	"sugarplan/controller/BaseController",
	"sugarplan/model/chart"
], function (BaseController, chart) {
	"use strict";

	/**
	 * Overview is the executive dashboard. Every figure on it comes from the
	 * dashboard endpoint, which in turn derives everything from the stored daily
	 * rows, so what is shown here reconciles with the transaction data by
	 * construction rather than by a separate reporting copy.
	 */
	return BaseController.extend("sugarplan.controller.Overview", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({
				versions: [], versionId: "",
				// One property per chart rather than one blob: a panel that is
				// collapsed still binds, and a single string would redraw all
				// eight every time any of them changed.
				chartHtml: "", dailyHtml: "", recoveryHtml: "",
				mixHtml: "", channelHtml: "", paretoHtml: "", calendarHtml: ""
			}), "view");
			this.getView().setModel(this.getService().newModel({
				alerts: [], storage: [], products: [], shipments: [],
				cane: {}, rawSugar: {}, downtime: {}
			}), "dash");

			this.getRouter().getRoute("overview").attachPatternMatched(this._onDisplay, this);
			this.onContextRefresh(this._onDisplay);
		},

		_onDisplay: function () {
			var sSeasonId = this.requireSeason();
			if (!sSeasonId) {
				return;
			}
			var that = this;
			this.setBusy(true);

			this.getService().listVersions(sSeasonId).then(function (oPage) {
				var aVersions = (oPage.value || []).filter(function (oVersion) {
					return oVersion.planType !== "ACTUAL";
				});
				that.getView().getModel("view").setProperty("/versions", aVersions);
				var sSelected = that.getView().getModel("view").getProperty("/versionId");
				if (!sSelected || !aVersions.some(function (v) { return v.id === sSelected; })) {
					// Prefer the released baseline; otherwise the latest version.
					var oReleased = aVersions.filter(function (v) { return v.status === "RELEASED"; })[0];
					sSelected = (oReleased || aVersions[aVersions.length - 1] || {}).id || "";
					that.getView().getModel("view").setProperty("/versionId", sSelected);
				}
				return that._loadDashboard(sSeasonId, sSelected);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		_loadDashboard: function (sSeasonId, sVersionId) {
			var that = this;
			return this.getService().dashboard(sSeasonId, sVersionId).then(function (oDashboard) {
				that.getView().getModel("dash").setData(oDashboard);
				that._drawCharts(oDashboard);
				that.setBusy(false);
			});
		},

		/**
		 * _drawCharts renders every visual from the one dashboard payload.
		 *
		 * They are drawn together from a single response so that no two charts on
		 * the page can be showing different moments of the same season.
		 */
		_drawCharts: function (oDashboard) {
			var oModel = this.getView().getModel("view");
			var aCane = oDashboard.caneTrend || [];

			oModel.setProperty("/chartHtml", chart.cumulativeCurve(aCane));
			oModel.setProperty("/dailyHtml",
				chart.dailyTrend(aCane, oDashboard.caneRollingAverage || []));
			oModel.setProperty("/calendarHtml", chart.calendar(aCane));

			// The band is the one the verdict is reached by, taken from the
			// dashboard rather than derived here, so the chart and the alert
			// cannot come to different conclusions about the same day.
			var oRaw = oDashboard.rawSugar || {};
			oModel.setProperty("/recoveryHtml", chart.recoveryTrend(
				oDashboard.recoveryTrend || [], oRaw.minRecoveryPct, oRaw.maxRecoveryPct));

			oModel.setProperty("/mixHtml", chart.productMix(oDashboard.productTrend || []));
			oModel.setProperty("/channelHtml", chart.channelTrend(oDashboard.shipmentTrend || []));
			oModel.setProperty("/paretoHtml",
				chart.downtimePareto((oDashboard.downtime || {}).byReason || []));
		},

		onVersionChange: function () {
			var sSeasonId = this.getAppModel().getProperty("/selectedSeasonId");
			var sVersionId = this.getView().getModel("view").getProperty("/versionId");
			var that = this;
			this.setBusy(true);
			this._loadDashboard(sSeasonId, sVersionId).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		onRefresh: function () {
			this._onDisplay();
		},

		onNavBack: function () {
			this.navTo("launchpad");
		},

		// --- small presentation formatters that need two values ------------

		formatOfTarget: function (sPattern, vTarget) {
			return this._fill(sPattern, this.formatter.tons0(vTarget));
		},

		formatAtRate: function (sPattern, vRate) {
			return this._fill(sPattern, this.formatter.tons0(vRate));
		},

		formatPlannedEnd: function (sPattern, sDate) {
			return this._fill(sPattern, this.formatter.date(sDate));
		},

		formatTargetRecovery: function (sPattern, vPct) {
			return this._fill(sPattern, this.formatter.percent(vPct));
		},

		formatRawSugar: function (sPattern, vTons) {
			return this._fill(sPattern, this.formatter.tons0(vTons));
		},

		_fill: function (sPattern, sValue) {
			if (!sPattern) {
				return sValue || "";
			}
			return sPattern.replace("{0}", sValue === undefined || sValue === null ? "" : sValue);
		}
	});
});
