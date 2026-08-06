sap.ui.define([
	"sugarplan/controller/BaseController"
], function (BaseController) {
	"use strict";

	/**
	 * Reports runs a report from the server catalogue and previews it, then
	 * hands the same report to Excel, CSV or PDF. Preview and export come from
	 * one server-side builder, so what is on screen is what is in the file.
	 */
	return BaseController.extend("sugarplan.controller.Reports", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({
				catalogue: [], versions: [], versionId: "",
				selectedCode: "", from: "", to: "",
				report: null, columns: [], rows: [], busy: false
			}), "view");
			this.getRouter().getRoute("reports").attachPatternMatched(this._onDisplay, this);
			this.onContextRefresh(this._onDisplay);
		},

		_onDisplay: function () {
			var sSeasonId = this.requireSeason();
			if (!sSeasonId) {
				return;
			}
			var that = this;
			this.setBusy(true);

			Promise.all([
				this.getService().listReports(),
				this.getService().listVersions(sSeasonId)
			]).then(function (aResults) {
				var oModel = that.getView().getModel("view");
				oModel.setProperty("/catalogue", aResults[0].value || []);

				var aVersions = (aResults[1].value || []).filter(function (oVersion) {
					return oVersion.planType !== "ACTUAL";
				});
				oModel.setProperty("/versions", aVersions);
				if (!oModel.getProperty("/versionId") && aVersions.length) {
					var oReleased = aVersions.filter(function (v) { return v.status === "RELEASED"; })[0];
					oModel.setProperty("/versionId", (oReleased || aVersions[aVersions.length - 1]).id);
				}
				that.setBusy(false);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		_params: function () {
			var oModel = this.getView().getModel("view");
			return {
				versionId: oModel.getProperty("/versionId"),
				seasonId: this.getAppModel().getProperty("/selectedSeasonId"),
				from: oModel.getProperty("/from"),
				to: oModel.getProperty("/to")
			};
		},

		onSelectReport: function (oEvent) {
			var oItem = oEvent.getParameter("listItem") || oEvent.getSource();
			var oContext = oItem.getBindingContext("view");
			if (!oContext) {
				return;
			}
			var oDefinition = oContext.getObject();
			this.getView().getModel("view").setProperty("/selectedCode", oDefinition.code);
			this.onRun();
		},

		onRun: function () {
			var oModel = this.getView().getModel("view");
			var sCode = oModel.getProperty("/selectedCode");
			if (!sCode) {
				this.showToast(this.getText("selectReportFirst"));
				return;
			}
			var that = this;
			this.setBusy(true);

			this.getService().runReport(sCode, this._params()).then(function (oReport) {
				oModel.setProperty("/report", oReport);
				oModel.setProperty("/columns", oReport.columns || []);
				// The preview binds rows as objects, because a UI5 table cannot
				// bind a bare array of arrays to named columns.
				oModel.setProperty("/rows", (oReport.rows || []).map(function (aCells) {
					var oRow = {};
					aCells.forEach(function (sCell, i) {
						oRow["c" + i] = sCell;
					});
					return oRow;
				}));
				that._buildPreviewColumns(oReport);
				that.setBusy(false);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		/** _buildPreviewColumns rebuilds the preview table for the report that
		 * was just run, because each report has its own column set. */
		_buildPreviewColumns: function (oReport) {
			var oTable = this.byId("previewTable");
			if (!oTable) {
				return;
			}
			oTable.destroyColumns();
			oTable.unbindItems();

			var Column = sap.ui.require("sap/m/Column");
			var Text = sap.ui.require("sap/m/Text");
			var ColumnListItem = sap.ui.require("sap/m/ColumnListItem");
			if (!Column || !Text || !ColumnListItem) {
				return;
			}

			var aCells = [];
			(oReport.columns || []).forEach(function (oColumn, i) {
				oTable.addColumn(new Column({
					hAlign: oColumn.align === "right" ? "End" : "Begin",
					header: new Text({ text: oColumn.header })
				}));
				aCells.push(new Text({ text: "{view>c" + i + "}" }));
			});

			oTable.bindItems({
				path: "view>/rows",
				templateShareable: false,
				template: new ColumnListItem({ cells: aCells })
			});
		},

		onExport: function (oEvent) {
			var sFormat = oEvent.getSource().data("format");
			var sCode = this.getView().getModel("view").getProperty("/selectedCode");
			if (!sCode) {
				this.showToast(this.getText("selectReportFirst"));
				return;
			}
			var that = this;
			this.setBusy(true);
			this.getService().downloadReport(sCode, sFormat, this._params())
				.then(function () {
					that.setBusy(false);
					that.showToast(that.getText("exportStarted", [sFormat.toUpperCase()]));
				})
				.catch(function (oProblem) {
					that.showError(oProblem);
				});
		},

		onNavBack: function () {
			this.navTo("launchpad");
		}
	});
});
