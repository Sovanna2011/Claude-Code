sap.ui.define([
	"sugarplan/controller/BaseController",
	"sap/m/Dialog",
	"sap/m/Button",
	"sap/m/Label",
	"sap/m/Input",
	"sap/m/Select",
	"sap/m/MultiComboBox",
	"sap/m/CheckBox",
	"sap/m/Text",
	"sap/m/VBox",
	"sap/ui/core/Item",
	"sap/m/Table",
	"sap/m/Column",
	"sap/m/ColumnListItem",
	"sap/m/ObjectNumber"
], function (BaseController, Dialog, Button, Label, Input, Select, MultiComboBox, CheckBox,
	Text, VBox, Item, Table, Column, ColumnListItem, ObjectNumber) {
	"use strict";

	return BaseController.extend("sugarplan.controller.SeasonPlans", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({ versions: [], all: [] }), "view");
			this.getRouter().getRoute("plans").attachPatternMatched(this._onDisplay, this);
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
				var aVersions = oPage.value || [];
				that.getView().getModel("view").setData({ versions: aVersions, all: aVersions });
				that.setBusy(false);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		onSearch: function (oEvent) {
			var sQuery = (oEvent.getParameter("newValue") || "").toLowerCase();
			var oModel = this.getView().getModel("view");
			var aAll = oModel.getProperty("/all") || [];
			oModel.setProperty("/versions", aAll.filter(function (oVersion) {
				return !sQuery ||
					(oVersion.code || "").toLowerCase().indexOf(sQuery) >= 0 ||
					(oVersion.description || "").toLowerCase().indexOf(sQuery) >= 0;
			}));
		},

		onOpenVersion: function (oEvent) {
			// The handler serves both the table's itemPress, whose source is the
			// table and whose row arrives as a parameter, and the row's own
			// press, whose source is the row itself.
			var oItem = oEvent.getParameter("listItem") || oEvent.getSource();
			var oContext = oItem.getBindingContext("view");
			if (!oContext) {
				return;
			}
			this.navTo("planDetail", { versionId: oContext.getProperty("id") });
		},

		onOpenBoard: function (oEvent) {
			var oContext = oEvent.getSource().getBindingContext("view");
			if (!oContext) {
				return;
			}
			this.navTo("board", { versionId: oContext.getProperty("id") });
		},

		onNavBack: function () {
			this.navTo("launchpad");
		},

		/**
		 * onNewScenario copies an existing version. This is the documented way
		 * to explore a what-if: copy the baseline, change an assumption, and
		 * regenerate, so the baseline itself is never disturbed.
		 */
		onNewScenario: function () {
			var that = this;
			var aVersions = this.getView().getModel("view").getProperty("/all") || [];
			if (!aVersions.length) {
				this.showToast(this.getText("noVersionsToCopy"));
				return;
			}

			var oSource = new Select({ width: "100%" });
			aVersions.forEach(function (oVersion) {
				oSource.addItem(new Item({ key: oVersion.id, text: oVersion.code + " — " + oVersion.description }));
			});
			var oCode = new Input({ width: "100%", placeholder: "WHATIF-1", required: true });
			var oDescription = new Input({ width: "100%", placeholder: this.getText("scenarioDescriptionHint") });
			var oType = new Select({ width: "100%" });
			[["WHATIF", "What-if simulation"], ["REVISED", "Revised plan"],
				["FORECAST", "Forecast"], ["BUDGET", "Budget"]].forEach(function (aPair) {
				oType.addItem(new Item({ key: aPair[0], text: aPair[1] }));
			});
			var oRecovery = new Input({ width: "100%", type: "Number",
				placeholder: this.getText("scenarioRecoveryHint") });
			var oCopyRows = new CheckBox({ selected: false, text: this.getText("scenarioCopyRows") });

			var oDialog = new Dialog({
				title: this.getText("newScenario"),
				contentWidth: "28rem",
				content: [new VBox({
					class: "sapUiSmallMargin",
					items: [
						new Text({ text: this.getText("scenarioIntro") }),
						new Label({ text: this.getText("scenarioSource"), labelFor: oSource }), oSource,
						new Label({ text: this.getText("scenarioCode"), required: true, labelFor: oCode }), oCode,
						new Label({ text: this.getText("scenarioDescription"), labelFor: oDescription }), oDescription,
						new Label({ text: this.getText("planType"), labelFor: oType }), oType,
						new Label({ text: this.getText("scenarioRecovery"), labelFor: oRecovery }), oRecovery,
						oCopyRows
					]
				})],
				beginButton: new Button({
					text: this.getText("create"), type: "Emphasized",
					press: function () {
						var sCode = (oCode.getValue() || "").trim();
						if (!sCode) {
							oCode.setValueState("Error");
							oCode.setValueStateText(that.getText("scenarioCodeRequired"));
							return;
						}
						var oPayload = {
							code: sCode,
							description: oDescription.getValue(),
							planType: oType.getSelectedKey(),
							copyDailyRows: oCopyRows.getSelected()
						};
						var sRecovery = (oRecovery.getValue() || "").trim();
						if (sRecovery) {
							oPayload.assumptionOverrides = { RAW_RECOVERY_PCT: parseFloat(sRecovery) };
						}
						oDialog.setBusy(true);
						that.getService().copyVersion(oSource.getSelectedKey(), oPayload)
							.then(function (oCreated) {
								oDialog.close();
								that.showToast(that.getText("scenarioCreated", [oCreated.code]));
								that.navTo("planDetail", { versionId: oCreated.id });
							})
							.catch(function (oProblem) {
								oDialog.setBusy(false);
								that.showError(oProblem);
							});
					}
				}),
				endButton: new Button({ text: this.getText("cancel"), press: function () { oDialog.close(); } }),
				afterClose: function () { oDialog.destroy(); }
			});
			this.getView().addDependent(oDialog);
			oDialog.open();
		},

		/** onCompare puts two versions side by side and, more usefully, shows
		 * which assumptions differ between them. */
		onCompare: function () {
			var that = this;
			var aVersions = this.getView().getModel("view").getProperty("/all") || [];
			if (aVersions.length < 2) {
				this.showToast(this.getText("needTwoVersions"));
				return;
			}

			// A review reads a budget, its what-ifs and the actuals so far in
			// columns. Two Selects could only ever ask half that question.
			var oPick = new MultiComboBox({ width: "100%" });
			var oBaseline = new Select({ width: "100%" });
			aVersions.forEach(function (oVersion) {
				var sText = oVersion.code + " — " + (oVersion.description || oVersion.planType);
				oPick.addItem(new Item({ key: oVersion.id, text: sText }));
				oBaseline.addItem(new Item({ key: oVersion.id, text: sText }));
			});
			// Open on the two ends of the list, which is the comparison somebody
			// almost always wants first: the baseline against the newest scenario.
			oPick.setSelectedKeys([aVersions[0].id, aVersions[aVersions.length - 1].id]);
			oBaseline.setSelectedKey(aVersions[0].id);

			var oActuals = new CheckBox({
				text: this.getText("compareIncludeActual"), selected: false
			});

			var oDimension = new Select({ width: "100%" });
			[["PROCESS", "By process stage"], ["DATE", "By date"], ["PRODUCT", "By product"],
				["WAREHOUSE", "By warehouse"], ["CHANNEL", "By shipment channel"]].forEach(function (aPair) {
				oDimension.addItem(new Item({ key: aPair[0], text: aPair[1] }));
			});

			var oResult = new VBox({ class: "sapUiSmallMarginTop" });

			var oDialog = new Dialog({
				title: this.getText("compareVersions"),
				contentWidth: "56rem",
				content: [new VBox({
					class: "sapUiSmallMargin",
					items: [
						new Label({ text: this.getText("compareVersionsPick") }), oPick,
						new Label({ text: this.getText("compareBaseline") }), oBaseline,
						oActuals,
						new Label({ text: this.getText("compareDimension") }), oDimension,
						oResult
					]
				})],
				beginButton: new Button({
					text: this.getText("compare"), type: "Emphasized",
					press: function () {
						var aPicked = oPick.getSelectedKeys();
						if (aPicked.length < 1) {
							that.showToast(that.getText("needTwoVersions"));
							return;
						}
						oDialog.setBusy(true);
						that.getService().compareMatrix({
							versionIds: aPicked,
							baselineVersionId: oBaseline.getSelectedKey(),
							includeActual: oActuals.getSelected(),
							dimension: oDimension.getSelectedKey()
						}).then(function (oComparison) {
							oDialog.setBusy(false);
							that._renderComparison(oResult, oComparison);
						}).catch(function (oProblem) {
							oDialog.setBusy(false);
							that.showError(oProblem);
						});
					}
				}),
				endButton: new Button({ text: this.getText("close"), press: function () { oDialog.close(); } }),
				afterClose: function () { oDialog.destroy(); }
			});
			this.getView().addDependent(oDialog);
			oDialog.open();
		},

		/**
		 * _renderComparison draws the matrix: one column per version.
		 *
		 * The baseline column is marked rather than left to be inferred from the
		 * zeros - a column of zero differences could equally be a version that
		 * happens to match, and those are different facts.
		 */
		_renderComparison: function (oContainer, oComparison) {
			var that = this;
			oContainer.destroyItems();

			var aColumns = oComparison.columns || [];
			var iBaseline = 0;
			aColumns.forEach(function (oCol, i) { if (oCol.isBaseline) { iBaseline = i; } });

			function matrixTable(sTitle, aRows, bAssumption) {
				if (!aRows || !aRows.length) {
					return null;
				}
				var oTable = new Table({ class: "sugarTable sapUiSmallMarginTop" });
				oTable.addColumn(new Column({ header: new Text({ text: sTitle }) }));
				aColumns.forEach(function (oCol, i) {
					var sHead = oCol.version.code +
						(oCol.series === "ACTUAL" ? " (actual)" : "") +
						(i === iBaseline ? " ·" : "");
					oTable.addColumn(new Column({ hAlign: "End", header: new Text({ text: sHead }) }));
				});

				aRows.forEach(function (oRow) {
					var aCells = [new Text({
						text: (bAssumption ? "" : oRow.measure + " · ") + oRow.label
					})];
					oRow.values.forEach(function (sValue, i) {
						if (i === iBaseline) {
							aCells.push(new ObjectNumber({
								number: bAssumption ? sValue : that.formatter.tons0(sValue),
								emphasized: true
							}));
							return;
						}
						var fDelta = parseFloat(oRow.deltas[i]) || 0;
						aCells.push(new ObjectNumber({
							number: bAssumption ? sValue : that.formatter.tons0(sValue),
							unit: (fDelta > 0 ? "+" : "") +
								(bAssumption ? oRow.deltas[i] : that.formatter.tons0(oRow.deltas[i])),
							state: fDelta === 0 ? "None" : (fDelta > 0 ? "Success" : "Warning")
						}));
					});
					oTable.addItem(new ColumnListItem({ cells: aCells }));
				});
				return oTable;
			}

			var oAssumptions = matrixTable(this.getText("compareAssumptions"),
				oComparison.assumptions, true);
			if (oAssumptions) {
				oContainer.addItem(oAssumptions);
			} else {
				oContainer.addItem(new Text({ text: this.getText("compareNoAssumptionChange") }));
			}

			var oTotals = matrixTable(this.getText("compareTotals"), oComparison.totals, false);
			if (oTotals) {
				oContainer.addItem(oTotals);
			}
			oContainer.addItem(new Text({
				text: this.getText("compareBaselineNote", [aColumns[iBaseline].version.code]),
				class: "sapUiSmallMarginTop"
			}));
		}
	});
});
