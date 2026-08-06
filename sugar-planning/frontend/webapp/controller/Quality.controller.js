sap.ui.define([
	"sugarplan/controller/BaseController"
], function (BaseController) {
	"use strict";

	/**
	 * Quality is the laboratory's page: samples taken, the sheet of
	 * measurements entered against them, and the stock a failure blocks.
	 *
	 * Judging a measurement is the API's job, not this page's. The sheet is sent
	 * as entered and comes back with each result already marked pass, warning or
	 * fail against the limits in force on the sample's business date, together
	 * with any parameter that had no limits at all.
	 */
	return BaseController.extend("sugarplan.controller.Quality", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({
				samples: [],
				holds: [],
				parameters: [],
				products: [],
				warehouses: [],
				openHoldsOnly: true,
				canWrite: false,
				canRelease: false,
				sample: { productId: "", businessDate: "", comment: "" },
				sheet: { sampleNo: "", results: [], complete: true, holdWarehouse: "", holdQuantity: "" }
			}), "view");
			this.getRouter().getRoute("quality").attachPatternMatched(this._onDisplay, this);
			this.onContextRefresh(this._onDisplay);
		},

		_onDisplay: function () {
			if (!this.getAppModel().getProperty("/signedIn")) {
				return;
			}
			var oModel = this.getView().getModel("view");
			var oService = this.getService();
			var that = this;

			oModel.setProperty("/canWrite", this.can("quality:write"));
			oModel.setProperty("/canRelease", this.can("quality:release"));
			this.setBusy(true);

			Promise.all([
				oService.listSamples({ $top: 100 }),
				oService.listHolds({ openOnly: oModel.getProperty("/openHoldsOnly") || undefined }),
				oService.listQualityParameters(),
				oService.listMaster("products", { active: "true" }),
				oService.listMaster("warehouses", { active: "true" })
			]).then(function (aResults) {
				oModel.setProperty("/samples", aResults[0].value || []);
				oModel.setProperty("/holds", aResults[1].value || []);
				oModel.setProperty("/parameters", aResults[2].value || []);
				oModel.setProperty("/products", aResults[3].value || []);
				oModel.setProperty("/warehouses", aResults[4].value || []);
				that.setBusy(false);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		onToggleOpenHolds: function (oEvent) {
			this.getView().getModel("view").setProperty("/openHoldsOnly", oEvent.getParameter("selected"));
			this._onDisplay();
		},

		_factoryId: function () {
			var oProfile = this.getService().getSessionProfile() || {};
			var aFactories = oProfile.factories || [];
			return aFactories.length === 1 ? aFactories[0] : "";
		},

		// ------------------------------------------------------------------
		// Samples
		// ------------------------------------------------------------------

		onOpenSample: function () {
			this.getView().getModel("view").setProperty("/sample", {
				productId: "",
				businessDate: new Date().toISOString().slice(0, 10),
				comment: ""
			});
			this._dialog("sugarplan.view.fragment.SampleDialog").then(function (oDialog) {
				oDialog.open();
			});
		},

		onCancelSample: function () {
			this._closeDialog("sugarplan.view.fragment.SampleDialog");
		},

		onSubmitSample: function () {
			var oSample = this.getView().getModel("view").getProperty("/sample");
			var that = this;

			if (!oSample.productId) {
				this.showError({
					title: this.getText("qualitySampleTitle"),
					detail: this.getText("qualityProductRequired")
				});
				return;
			}

			this.setBusy(true);
			this.getService().createSample({
				productId: oSample.productId,
				factoryId: this._factoryId(),
				businessDate: oSample.businessDate,
				comment: oSample.comment
			}).then(function (oCreated) {
				that._closeDialog("sugarplan.view.fragment.SampleDialog");
				that.showToast(that.getText("qualitySampleCreated", [oCreated.sampleNo]));
				that._onDisplay();
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		/** onCertificate downloads the certificate of analysis for a completed
		 * sample. The PDF is what is sent with a consignment. */
		onCertificate: function (oEvent) {
			var oSample = oEvent.getSource().getBindingContext("view").getObject();
			var that = this;
			this.setBusy(true);
			this.getService().downloadCertificate(oSample.id, oSample.sampleNo, "pdf")
				.then(function () {
					that.setBusy(false);
					that.showToast(that.getText("certificateIssued", [oSample.sampleNo]));
				}).catch(function (oProblem) {
					that.showError(oProblem);
				});
		},

		/** onEnterResults opens the sheet with one empty row per configured
		 * parameter, so the laboratory fills in what it measured rather than
		 * choosing parameters from a list first. */
		onEnterResults: function (oEvent) {
			var oItem = oEvent.getParameter("listItem") || oEvent.getSource();
			var oContext = oItem.getBindingContext("view");
			if (!oContext) {
				return;
			}
			var oSample = oContext.getObject();
			var oModel = this.getView().getModel("view");

			if (oSample.status !== "OPEN") {
				this._showSample(oSample);
				return;
			}

			var aExisting = oSample.results || [];
			var aRows = (oModel.getProperty("/parameters") || []).map(function (oParameter) {
				var oPrevious = aExisting.filter(function (oResult) {
					return oResult.parameterId === oParameter.id;
				})[0];
				return {
					parameterId: oParameter.id,
					code: oParameter.code,
					name: oParameter.name,
					uom: oParameter.uom,
					value: oPrevious ? oPrevious.value : ""
				};
			});

			oModel.setProperty("/sheet", {
				sampleId: oSample.id,
				sampleNo: oSample.sampleNo,
				results: aRows,
				complete: true,
				holdWarehouse: "",
				holdQuantity: ""
			});
			this._dialog("sugarplan.view.fragment.ResultsDialog").then(function (oDialog) {
				oDialog.open();
			});
		},

		/** _showSample displays a completed sample read-only: its results are
		 * history, and history is not re-entered. */
		_showSample: function (oSample) {
			var aLines = (oSample.results || []).map(function (oResult) {
				return oResult.parameterId + ": " + oResult.value + " (" + oResult.status + ")";
			});
			this.showError({
				title: this.getText("qualitySampleComplete", [oSample.sampleNo]),
				detail: aLines.join("\n") || this.getText("qualityNoResults")
			});
		},

		onCancelResults: function () {
			this._closeDialog("sugarplan.view.fragment.ResultsDialog");
		},

		onSubmitResults: function () {
			var oModel = this.getView().getModel("view");
			var oSheet = oModel.getProperty("/sheet");
			var that = this;

			// A parameter left blank was not measured, so it is not sent: an
			// empty box is not a reading of zero.
			var aResults = (oSheet.results || []).filter(function (oRow) {
				return oRow.value !== "" && oRow.value !== null && oRow.value !== undefined;
			}).map(function (oRow) {
				return { parameterId: oRow.parameterId, value: String(oRow.value), uom: oRow.uom };
			});
			if (!aResults.length) {
				this.showError({
					title: this.getText("qualityResultsTitle"),
					detail: this.getText("qualityResultsRequired")
				});
				return;
			}

			this.setBusy(true);
			this.getService().recordResults(oSheet.sampleId, {
				results: aResults,
				complete: !!oSheet.complete,
				holdWarehouse: oSheet.holdWarehouse || undefined,
				holdQuantity: oSheet.holdQuantity ? String(oSheet.holdQuantity) : undefined
			}).then(function (oOutcome) {
				that._closeDialog("sugarplan.view.fragment.ResultsDialog");
				that.showToast(that.getText("qualityVerdict",
					[oSheet.sampleNo, oOutcome.verdict]));

				// A parameter with no specification in force judged nothing.
				// Saying so is the point: silence is how a limit goes years
				// without being set.
				if (oOutcome.unspecified && oOutcome.unspecified.length) {
					that.showError({
						title: that.getText("qualityUnspecifiedTitle"),
						detail: that.getText("qualityUnspecified",
							[oOutcome.unspecified.join(", ")])
					});
				}
				that._onDisplay();
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		// ------------------------------------------------------------------
		// Holds
		// ------------------------------------------------------------------

		onReleaseHold: function (oEvent) {
			var oHold = oEvent.getSource().getBindingContext("view").getObject();
			var that = this;

			this.confirm(this.getText("qualityReleaseConfirm",
				[oHold.quantity]), this.getText("qualityReleaseTitle")).then(function (bOk) {
				if (!bOk) {
					return;
				}
				that.setBusy(true);
				that.getService().releaseHold(oHold.id, {
					reason: that.getText("qualityReleaseReason"),
					rowVersion: oHold.rowVersion
				}).then(function () {
					that.showToast(that.getText("qualityReleased"));
					that._onDisplay();
				}).catch(function (oProblem) {
					that.showError(oProblem);
				});
			});
		},

		onRefresh: function () {
			this._onDisplay();
		},

		onNavBack: function () {
			this.navTo("launchpad");
		}
	});
});
