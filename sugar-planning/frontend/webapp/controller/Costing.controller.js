sap.ui.define([
	"sugarplan/controller/BaseController",
	"sap/ui/core/Fragment",
	"sugarplan/model/chart"
], function (BaseController, Fragment, chart) {
	"use strict";

	/**
	 * Costing is the controller's page: what a ton of sugar costs, and why the
	 * actual differs from the plan.
	 *
	 * The variance is shown split into the part caused by paying a different
	 * price and the part caused by using a different quantity, because those are
	 * two different conversations with two different people. The two halves
	 * always add up to the total; the API guarantees it.
	 */
	return BaseController.extend("sugarplan.controller.Costing", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({
				lines: [],
				totals: null,
				plannedDrivers: null,
				actualDrivers: null,
				warnings: [],
				runs: [],
				elements: [],
				rates: [],
				currency: "",
				from: "",
				to: "",
				canWrite: false,
				chartHtml: "",
				rate: this._emptyRate()
			}), "view");
			this.getRouter().getRoute("costing").attachPatternMatched(this._onDisplay, this);
			this.onContextRefresh(this._onDisplay);
		},

		_emptyRate: function () {
			return {
				elementId: "", rateType: "STANDARD", rate: "",
				currency: "USD", validFrom: "", note: ""
			};
		},

		_onDisplay: function () {
			var sSeasonId = this.requireSeason();
			if (!sSeasonId) {
				return;
			}
			var oModel = this.getView().getModel("view");
			var oService = this.getService();
			var that = this;

			oModel.setProperty("/canWrite", this.can("cost:write"));
			this.setBusy(true);

			Promise.all([
				oService.costRun({ seasonId: sSeasonId }),
				oService.listCostRuns({ seasonId: sSeasonId }),
				oService.listCostElements()
			]).then(function (aResults) {
				that._applyRun(aResults[0]);
				oModel.setProperty("/runs", aResults[1].value || []);
				oModel.setProperty("/elements", aResults[2].value || []);
				that.setBusy(false);
			}).catch(function (oProblem) {
				if (oProblem && oProblem.status === 404) {
					// A season with no plan version yet is not an error worth a
					// dialog; the page simply has nothing to cost.
					that.setBusy(false);
					return;
				}
				that.showError(oProblem);
			});
		},

		_applyRun: function (oResult) {
			var oModel = this.getView().getModel("view");
			oModel.setProperty("/lines", oResult.lines || []);
			oModel.setProperty("/totals", oResult.totals || null);
			oModel.setProperty("/plannedDrivers", oResult.plannedDrivers || null);
			oModel.setProperty("/actualDrivers", oResult.actualDrivers || null);
			oModel.setProperty("/warnings", oResult.warnings || []);
			oModel.setProperty("/currency", oResult.currency || "");
			oModel.setProperty("/from", oResult.from || "");
			oModel.setProperty("/to", oResult.to || "");
			oModel.setProperty("/chartHtml", chart.costByCategory(oResult.lines || []));
		},

		/** onRerun costs the range the user has narrowed to. */
		onRerun: function () {
			var oModel = this.getView().getModel("view");
			var that = this;

			this.setBusy(true);
			this.getService().costRun({
				seasonId: this.requireSeason(),
				from: oModel.getProperty("/from") || undefined,
				to: oModel.getProperty("/to") || undefined
			}).then(function (oResult) {
				that._applyRun(oResult);
				that.setBusy(false);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		/** onSaveRun stores the figure so a board pack can quote it and it can
		 * still be reproduced after the rates have moved on. */
		onSaveRun: function () {
			var oModel = this.getView().getModel("view");
			var that = this;
			var sCode = (oModel.getProperty("/from") || "") + "_" + (oModel.getProperty("/to") || "");

			this.setBusy(true);
			this.getService().costRun({
				seasonId: this.requireSeason(),
				from: oModel.getProperty("/from") || undefined,
				to: oModel.getProperty("/to") || undefined,
				save: true, code: sCode
			}).then(function (oResult) {
				that.showToast(that.getText("costingSaved", [oResult.run.code]));
				that._onDisplay();
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		/** onOpenRun re-reads a stored run, which keeps the rates it used. */
		onOpenRun: function (oEvent) {
			var oItem = oEvent.getParameter("listItem") || oEvent.getSource();
			var oContext = oItem.getBindingContext("view");
			if (!oContext) {
				return;
			}
			var that = this;
			this.setBusy(true);
			this.getService().getCostRun(oContext.getObject().id).then(function (oRun) {
				that._applyRun({
					lines: oRun.lines, totals: oRun.totals,
					currency: oRun.currency, from: oRun.from, to: oRun.to
				});
				that.showToast(that.getText("costingOpened", [oRun.code]));
				that.setBusy(false);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		// ------------------------------------------------------------------
		// Rates
		// ------------------------------------------------------------------

		onOpenRates: function () {
			var oModel = this.getView().getModel("view");
			var that = this;

			this.setBusy(true);
			this.getService().listCostRates({ factoryId: this._factoryId() })
				.then(function (oPage) {
					// The rate carries an element id, which means nothing on a
					// screen, so the element's code and name are put on the row.
					var mElements = {};
					(oModel.getProperty("/elements") || []).forEach(function (oElement) {
						mElements[oElement.id] = oElement;
					});
					oModel.setProperty("/rates", (oPage.value || []).map(function (oRate) {
						var oElement = mElements[oRate.elementId];
						oRate.elementCode = oElement ? oElement.code : "";
						oRate.elementName = oElement ? oElement.name : oRate.elementId;
						return oRate;
					}));
					oModel.setProperty("/rate", that._emptyRate());
					that.setBusy(false);
					return that._dialog("sugarplan.view.fragment.CostRatesDialog");
				}).then(function (oDialog) {
					oDialog.open();
				}).catch(function (oProblem) {
					that.showError(oProblem);
				});
		},

		onCloseRates: function () {
			this._closeDialog("sugarplan.view.fragment.CostRatesDialog");
		},

		onSaveRate: function () {
			var oModel = this.getView().getModel("view");
			var oRate = oModel.getProperty("/rate");
			var that = this;

			if (!oRate.elementId || !oRate.rate || !oRate.validFrom) {
				this.showError({
					title: this.getText("costingRateIncompleteTitle"),
					detail: this.getText("costingRateIncomplete")
				});
				return;
			}

			this.setBusy(true);
			this.getService().saveCostRate({
				elementId: oRate.elementId,
				factoryId: this._factoryId(),
				rateType: oRate.rateType,
				rate: String(oRate.rate),
				currency: oRate.currency,
				validFrom: oRate.validFrom,
				note: oRate.note
			}).then(function () {
				that.showToast(that.getText("costingRateSaved"));
				that._closeDialog("sugarplan.view.fragment.CostRatesDialog");
				that._onDisplay();
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		_factoryId: function () {
			var oProfile = this.getService().getSessionProfile() || {};
			var aFactories = oProfile.factories || [];
			return aFactories.length === 1 ? aFactories[0] : "";
		},

		/** costVarianceState colours a money variance the way a controller reads
		 * it: spending more than the plan is bad news whatever the sign
		 * convention says. Named apart from the formatter's varianceState, which
		 * grades a tonnage against plan and means the opposite. */
		costVarianceState: function (vValue) {
			var fValue = parseFloat(vValue);
			if (isNaN(fValue) || fValue === 0) {
				return "None";
			}
			return fValue > 0 ? "Error" : "Success";
		},

		onRefresh: function () {
			this._onDisplay();
		},

		onNavBack: function () {
			this.navTo("launchpad");
		},

		_dialog: function (sName) {
			this._dialogs = this._dialogs || {};
			if (this._dialogs[sName]) {
				return Promise.resolve(this._dialogs[sName]);
			}
			var that = this;
			return Fragment.load({
				id: this.getView().getId(), name: sName, controller: this
			}).then(function (oDialog) {
				that.getView().addDependent(oDialog);
				that._dialogs[sName] = oDialog;
				return oDialog;
			});
		},

		_closeDialog: function (sName) {
			if (this._dialogs && this._dialogs[sName]) {
				this._dialogs[sName].close();
			}
			this.setBusy(false);
		}
	});
});
