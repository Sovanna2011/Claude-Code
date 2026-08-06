sap.ui.define([
	"sugarplan/controller/BaseController",
	"sap/m/Dialog",
	"sap/m/Button",
	"sap/m/Label",
	"sap/m/TextArea",
	"sap/m/DatePicker",
	"sap/m/Text",
	"sap/m/VBox"
], function (BaseController, Dialog, Button, Label, TextArea, DatePicker, Text, VBox) {
	"use strict";

	/**
	 * PlanDetail is the object page for one planning version: its assumptions,
	 * its product mix and the workflow actions available on it.
	 */
	return BaseController.extend("sugarplan.controller.PlanDetail", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({
				version: {}, assumptions: [], productMix: [],
				allowedActions: [], editable: false, warnings: [], summary: null
			}), "view");
			this.getRouter().getRoute("planDetail").attachPatternMatched(this._onDisplay, this);
		},

		_onDisplay: function (oEvent) {
			this._sVersionId = oEvent.getParameter("arguments").versionId;
			this._load();
		},

		_load: function () {
			var that = this;
			this.setBusy(true);

			Promise.all([
				this.getService().getVersion(this._sVersionId),
				// Names for the mix table; the API returns ids, and a planner
				// needs to read product and warehouse names.
				this.getService().listMaster("products"),
				this.getService().listMaster("warehouses"),
				this.getService().listMaster("packaging-types")
			]).then(function (aResults) {
				var oDetail = aResults[0];
				var mProducts = that._index(aResults[1].value);
				var mWarehouses = that._index(aResults[2].value);
				var mPackaging = that._index(aResults[3].value);

				var aMix = (oDetail.productMix || []).map(function (oEntry) {
					return Object.assign({}, oEntry, {
						productName: (mProducts[oEntry.productId] || {}).name || oEntry.productId,
						warehouseName: (mWarehouses[oEntry.warehouseId] || {}).name || "—",
						packagingName: (mPackaging[oEntry.packagingId] || {}).name || "—"
					});
				});

				that._sEtag = oDetail.__etag;
				that.getView().getModel("view").setData({
					version: oDetail.version,
					assumptions: oDetail.assumptions || [],
					productMix: aMix,
					allowedActions: oDetail.allowedActions || [],
					editable: oDetail.editable,
					warnings: that.getView().getModel("view").getProperty("/warnings") || [],
					summary: that.getView().getModel("view").getProperty("/summary")
				});
				that.setBusy(false);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		_index: function (aItems) {
			var mIndex = {};
			(aItems || []).forEach(function (oItem) {
				mIndex[oItem.id] = oItem;
			});
			return mIndex;
		},

		onNavBack: function () {
			this.navTo("plans");
		},

		onOpenBoard: function () {
			this.navTo("board", { versionId: this._sVersionId });
		},

		/** onAssumptionChange saves one assumption immediately, because a
		 * planner adjusting a rate expects the next generate run to use it. */
		onAssumptionChange: function (oEvent) {
			var oContext = oEvent.getSource().getBindingContext("view");
			if (!oContext) {
				return;
			}
			var oAssumption = oContext.getObject();
			var that = this;

			var fValue = parseFloat(oEvent.getSource().getValue());
			if (isNaN(fValue)) {
				oEvent.getSource().setValueState("Error");
				oEvent.getSource().setValueStateText(this.getText("assumptionNumeric"));
				return;
			}
			oEvent.getSource().setValueState("None");

			this.getService().saveAssumption(this._sVersionId, {
				code: oAssumption.code,
				description: oAssumption.description,
				value: fValue,
				uom: oAssumption.uom,
				validFrom: oAssumption.validFrom,
				validTo: oAssumption.validTo
			}).then(function () {
				that.showToast(that.getText("assumptionSaved", [oAssumption.code]));
			}).catch(function (oProblem) {
				that.showError(oProblem);
				that._load();
			});
		},

		/** onGenerate rebuilds the daily plan and reports what the generator
		 * found: capacity breaches, supply shortfalls and rate problems. */
		onGenerate: function () {
			var that = this;
			this.confirm(this.getText("generateConfirm"), this.getText("generatePlan"))
				.then(function (bConfirmed) {
					if (!bConfirmed) {
						return;
					}
					that.setBusy(true);
					return that.getService().generate(that._sVersionId, true).then(function (oResult) {
						that.getView().getModel("view").setProperty("/warnings", oResult.warnings || []);
						that.getView().getModel("view").setProperty("/summary", oResult.summary);
						that.setBusy(false);
						that.showToast(that.getText("generateDone", [
							oResult.summary.workingDays,
							that.formatter.tons0(oResult.summary.caneAllocatedTons)
						]));
					});
				})
				.catch(function (oProblem) {
					that.showError(oProblem);
				});
		},

		/**
		 * onAction runs a workflow transition. Actions that change history ask
		 * for a reason, which is stored on the audit record.
		 */
		onAction: function (oEvent) {
			var sAction = oEvent.getSource().getBindingContext("view").getObject();
			var bNeedsReason = ["REJECT", "REOPEN", "SUPERSEDE", "CLOSE"].indexOf(sAction) >= 0;
			var bIsRelease = sAction === "RELEASE";
			var that = this;

			if (!bNeedsReason && !bIsRelease) {
				this._runAction(sAction, {});
				return;
			}

			var oReason = new TextArea({ width: "100%", rows: 3,
				placeholder: this.getText("reasonPlaceholder") });
			var oLockThrough = new DatePicker({ width: "100%", valueFormat: "yyyy-MM-dd",
				displayFormat: "yyyy-MM-dd", visible: bIsRelease });

			var aContent = [new Text({ text: this.getText(bIsRelease ? "releaseIntro" : "reasonIntro") })];
			if (bIsRelease) {
				aContent.push(new Label({ text: this.getText("lockThrough"), labelFor: oLockThrough }), oLockThrough);
			}
			if (bNeedsReason) {
				aContent.push(new Label({ text: this.getText("reason"), required: true, labelFor: oReason }), oReason);
			}

			var oDialog = new Dialog({
				title: this.formatter.actionText(sAction),
				contentWidth: "26rem",
				content: [new VBox({ class: "sapUiSmallMargin", items: aContent })],
				beginButton: new Button({
					text: this.formatter.actionText(sAction), type: "Emphasized",
					press: function () {
						if (bNeedsReason && !(oReason.getValue() || "").trim()) {
							oReason.setValueState("Error");
							oReason.setValueStateText(that.getText("reasonRequired"));
							return;
						}
						var oPayload = { action: sAction };
						if (bNeedsReason) {
							oPayload.reason = oReason.getValue().trim();
						}
						if (bIsRelease && oLockThrough.getValue()) {
							oPayload.lockThrough = oLockThrough.getValue();
						}
						oDialog.close();
						that._runAction(sAction, oPayload);
					}
				}),
				endButton: new Button({ text: this.getText("cancel"), press: function () { oDialog.close(); } }),
				afterClose: function () { oDialog.destroy(); }
			});
			this.getView().addDependent(oDialog);
			oDialog.open();
		},

		_runAction: function (sAction, oPayload) {
			var that = this;
			this.setBusy(true);
			this.getService().transition(this._sVersionId,
				Object.assign({ action: sAction }, oPayload), this._sEtag)
				.then(function (oVersion) {
					that.showToast(that.getText("actionDone", [
						that.formatter.actionText(sAction),
						that.formatter.statusText(oVersion.status)
					]));
					that._load();
				})
				.catch(function (oProblem) {
					that.showError(oProblem);
					// A concurrency conflict means somebody else moved the plan
					// on; reload so the user sees the real state.
					if (oProblem && oProblem.status === 412) {
						that._load();
					}
				});
		}
	});
});
