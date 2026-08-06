sap.ui.define([
	"sugarplan/controller/BaseController"
], function (BaseController) {
	"use strict";

	/**
	 * Import is the screen that replaces the spreadsheet, one file at a time.
	 *
	 * It is deliberately three steps rather than one button. A file is uploaded
	 * and *staged*: nothing reaches the plan until somebody has looked at what
	 * would be written and said so. That is the whole point - a spreadsheet is
	 * where the bad data comes from, and an import that wrote straight through
	 * would be a way past every rule the rest of the system enforces.
	 */
	return BaseController.extend("sugarplan.controller.Import", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({
				mappings: [],
				mapping: "",
				allVersions: [],
				versions: [],
				versionId: "",
				series: "PLAN",
				fileName: "",
				job: null,
				rows: [],
				unmapped: [],
				missing: [],
				errorsOnly: false,
				history: [],
				canWrite: false
			}), "view");
			this.getRouter().getRoute("import").attachPatternMatched(this._onDisplay, this);
			this.onContextRefresh(this._onDisplay);
		},

		_onDisplay: function () {
			var sSeasonId = this.requireSeason();
			if (!sSeasonId) {
				return;
			}
			var oModel = this.getView().getModel("view");
			oModel.setProperty("/canWrite", this.can("plan:write") || this.can("actual:cane") ||
				this.can("actual:production") || this.can("actual:shipment"));

			var that = this;
			this.setBusy(true);
			Promise.all([
				this.getService().listImportMappings(),
				this.getService().listVersions(sSeasonId),
				this.getService().listImports()
			]).then(function (aResults) {
				var aMappings = aResults[0].value || [];
				oModel.setProperty("/mappings", aMappings);
				if (!oModel.getProperty("/mapping") && aMappings.length) {
					oModel.setProperty("/mapping", aMappings[0].code);
				}

				oModel.setProperty("/allVersions", aResults[1].value || []);
				that._applySeries();

				oModel.setProperty("/history", aResults[2].value || []);
				that.setBusy(false);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		/**
		 * onSeriesChange narrows the versions to the ones the series can be
		 * written to.
		 *
		 * Actuals live in their own container and plan figures never go there, so
		 * offering both lists together is offering a mistake: a file of actuals
		 * loaded into the plan would overwrite the baseline somebody is measuring
		 * against, and plan rows in the actuals container would claim the factory
		 * did something it has not done.
		 */
		onSeriesChange: function () {
			this._applySeries();
		},

		_applySeries: function () {
			var oModel = this.getView().getModel("view");
			var bActual = oModel.getProperty("/series") === "ACTUAL";
			var aVersions = (oModel.getProperty("/allVersions") || []).filter(function (oVersion) {
				return bActual ? oVersion.planType === "ACTUAL" : oVersion.planType !== "ACTUAL";
			});
			oModel.setProperty("/versions", aVersions);

			var sSelected = oModel.getProperty("/versionId");
			var bStillThere = aVersions.some(function (oVersion) { return oVersion.id === sSelected; });
			if (!bStillThere) {
				oModel.setProperty("/versionId", aVersions.length ? aVersions[0].id : "");
			}
		},

		/** onFileChange stages the file as soon as it is chosen: a preview is
		 * read-only, so there is nothing to confirm before showing one. */
		onFileChange: function (oEvent) {
			var oFile = oEvent.getParameter("files") && oEvent.getParameter("files")[0];
			if (!oFile) {
				return;
			}
			var oModel = this.getView().getModel("view");
			var that = this;

			this.setBusy(true);
			this.getService().stageImport(oFile, {
				mapping: oModel.getProperty("/mapping"),
				versionId: oModel.getProperty("/versionId"),
				series: oModel.getProperty("/series")
			}).then(function (oPreview) {
				that.setBusy(false);
				that._show(oPreview);
				oModel.setProperty("/fileName", oFile.name);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});

			// The picker is cleared so the same file can be chosen again after a
			// correction; a browser fires no change event for an unchanged value.
			oEvent.getSource().clear();
		},

		_show: function (oPreview) {
			var oModel = this.getView().getModel("view");
			oModel.setProperty("/job", oPreview.job || null);
			oModel.setProperty("/rows", oPreview.rows || []);
			oModel.setProperty("/unmapped", oPreview.unmapped || []);
			oModel.setProperty("/missing", oPreview.missing || []);
		},

		onErrorsOnlyChange: function () {
			var oModel = this.getView().getModel("view");
			var oJob = oModel.getProperty("/job");
			if (!oJob) {
				return;
			}
			var that = this;
			this.setBusy(true);
			this.getService().getImport(oJob.id, oModel.getProperty("/errorsOnly"))
				.then(function (oPreview) {
					that.setBusy(false);
					that._show(oPreview);
				}).catch(function (oProblem) {
					that.showError(oProblem);
				});
		},

		onDownloadErrors: function () {
			var oJob = this.getView().getModel("view").getProperty("/job");
			var that = this;
			this.getService().downloadImportErrors(oJob.id, oJob.fileName)
				.catch(function (oProblem) {
					that.showError(oProblem);
				});
		},

		/** onCommit writes the staged rows. Replacing rows that already exist is
		 * the normal case, so the confirmation says how many rather than warning
		 * about it. */
		onCommit: function () {
			var oModel = this.getView().getModel("view");
			var oJob = oModel.getProperty("/job");
			// Only the rows that will actually be written count as replacements:
			// a refused row replaces nothing, and saying otherwise would
			// overstate what the button is about to do.
			var iReplacing = (oModel.getProperty("/rows") || []).filter(function (oRow) {
				return oRow.replaces && !(oRow.errors && oRow.errors.length);
			}).length;
			var that = this;

			this.confirm(this.getText("importCommitConfirm",
				[oJob.validRows, iReplacing]), this.getText("importCommitTitle"))
				.then(function (bYes) {
					if (!bYes) {
						return;
					}
					that.setBusy(true);
					// Partial is sent when the file has bad rows, because the
					// confirmation the user just read said so.
					that.getService().commitImport(oJob.id, oJob.errorRows > 0)
						.then(function (oResult) {
							that.setBusy(false);
							that.showToast(that.getText("importCommitted",
								[oResult.written, oResult.replaced, oResult.skipped]));
							oModel.setProperty("/job", oResult.job);
							that._onDisplay();
						}).catch(function (oProblem) {
							that.showError(oProblem);
						});
				});
		},

		onCancelImport: function () {
			var oModel = this.getView().getModel("view");
			var oJob = oModel.getProperty("/job");
			var that = this;
			this.setBusy(true);
			this.getService().cancelImport(oJob.id).then(function (oCancelled) {
				that.setBusy(false);
				oModel.setProperty("/job", oCancelled);
				oModel.setProperty("/rows", []);
				that.showToast(that.getText("importCancelled"));
				that._onDisplay();
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		/** onOpenJob reopens an earlier upload from the history. */
		onOpenJob: function (oEvent) {
			var oJob = oEvent.getParameter("listItem").getBindingContext("view").getObject();
			var that = this;
			this.setBusy(true);
			this.getService().getImport(oJob.id, false).then(function (oPreview) {
				that.setBusy(false);
				that._show(oPreview);
				that.getView().getModel("view").setProperty("/fileName", oJob.fileName);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		onNavBack: function () {
			this.navTo("launchpad");
		}
	});
});
