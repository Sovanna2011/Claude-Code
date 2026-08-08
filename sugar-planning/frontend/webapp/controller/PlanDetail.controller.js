sap.ui.define([
	"sugarplan/controller/BaseController",
	"sap/m/Dialog",
	"sap/m/Button",
	"sap/m/Label",
	"sap/m/TextArea",
	"sap/m/DatePicker",
	"sap/m/Input",
	"sap/m/Select",
	"sap/m/Text",
	"sap/m/VBox",
	"sap/ui/core/Item"
], function (BaseController, Dialog, Button, Label, TextArea, DatePicker, Input, Select,
	Text, VBox, Item) {
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

				that._masters = {
					products: (aResults[1].value || []).filter(function (p) { return p.isFinished; }),
					warehouses: (aResults[2].value || []).filter(function (w) {
						return w.storageClass === "FINISHED";
					}),
					packaging: aResults[3].value || []
				};
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

		/**
		 * onEditMix opens the product mix entry form.
		 *
		 * The mix is the planning input the whole generator runs on: how much of
		 * each finished product the season is meant to make, and at what daily
		 * rate. It was readable and not enterable, so the only way a site could
		 * set it was to have it seeded.
		 *
		 * The daily rate matters as much as the tonnage. Left at zero the
		 * generator spreads the season evenly, which is a different plan: the
		 * mill runs its refinery at 400 t a day, and the refinery does not slow
		 * down because the arithmetic would be tidier.
		 */
		onEditMix: function (oEvent) {
			var oExisting = oEvent && oEvent.getSource && oEvent.getSource().getBindingContext("view")
				? oEvent.getSource().getBindingContext("view").getObject() : null;
			this._openMixDialog(oExisting);
		},

		onAddMix: function () {
			this._openMixDialog(null);
		},

		_openMixDialog: function (oEntry) {
			var that = this;
			var m = this._masters || { products: [], warehouses: [], packaging: [] };

			var oProduct = new Select({ width: "100%" });
			m.products.forEach(function (p) {
				oProduct.addItem(new Item({ key: p.id, text: p.code + " — " + p.name }));
			});
			var oPackaging = new Select({ width: "100%", forceSelection: false });
			oPackaging.addItem(new Item({ key: "", text: "—" }));
			m.packaging.forEach(function (p) {
				oPackaging.addItem(new Item({ key: p.id, text: p.code + " — " + p.name }));
			});
			var oWarehouse = new Select({ width: "100%", forceSelection: false });
			oWarehouse.addItem(new Item({ key: "", text: "—" }));
			m.warehouses.forEach(function (w) {
				oWarehouse.addItem(new Item({ key: w.id, text: w.code + " — " + w.name }));
			});

			var oTons = new Input({ width: "100%", type: "Number", placeholder: "106700" });
			var oRate = new Input({ width: "100%", type: "Number",
				placeholder: this.getText("mixRateHint") });

			if (oEntry) {
				oProduct.setSelectedKey(oEntry.productId);
				oPackaging.setSelectedKey(oEntry.packagingId || "");
				oWarehouse.setSelectedKey(oEntry.warehouseId || "");
				oTons.setValue(String(oEntry.seasonTons));
				if (parseFloat(oEntry.dailyRateTons) > 0) {
					oRate.setValue(String(oEntry.dailyRateTons));
				}
			}

			var oDialog = new Dialog({
				title: this.getText(oEntry ? "mixEdit" : "mixAdd"),
				contentWidth: "30rem",
				content: [new VBox({
					class: "sapUiSmallMargin",
					items: [
						new Label({ text: this.getText("product"), required: true }), oProduct,
						new Label({ text: this.getText("packaging") }), oPackaging,
						new Label({ text: this.getText("warehouse") }), oWarehouse,
						new Label({ text: this.getText("seasonTons"), required: true }), oTons,
						new Label({ text: this.getText("dailyRate") }), oRate,
						new Text({ text: this.getText("mixRateExplain"), class: "sapUiTinyMarginTop" })
					]
				})],
				beginButton: new Button({
					text: this.getText("save"), type: "Emphasized",
					press: function () {
						var fTons = parseFloat(oTons.getValue());
						if (isNaN(fTons) || fTons < 0) {
							oTons.setValueState("Error");
							oTons.setValueStateText(that.getText("mixTonsRequired"));
							return;
						}
						oTons.setValueState("None");
						var sRate = (oRate.getValue() || "").trim();
						var fRate = sRate ? parseFloat(sRate) : 0;
						if (sRate && (isNaN(fRate) || fRate < 0)) {
							oRate.setValueState("Error");
							oRate.setValueStateText(that.getText("mixRateInvalid"));
							return;
						}
						oRate.setValueState("None");

						oDialog.setBusy(true);
						that.getService().saveMixEntry(that._sVersionId, {
							id: oEntry ? oEntry.id : undefined,
							versionId: that._sVersionId,
							productId: oProduct.getSelectedKey(),
							packagingId: oPackaging.getSelectedKey() || undefined,
							warehouseId: oWarehouse.getSelectedKey() || undefined,
							seasonTons: String(fTons),
							dailyRateTons: String(fRate)
						}).then(function () {
							oDialog.close();
							that.showToast(that.getText("mixSaved"));
							that._load();
						}).catch(function (oProblem) {
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

		/** onDeleteMix removes a line. The plan has to be regenerated afterwards
		 * for the daily rows to stop reflecting it, which the toast says. */
		onDeleteMix: function (oEvent) {
			var oEntry = oEvent.getSource().getBindingContext("view").getObject();
			var that = this;
			this.confirm(this.getText("mixDeleteConfirm", [oEntry.productName]),
				this.getText("mixDelete")).then(function (bConfirmed) {
				if (!bConfirmed) {
					return;
				}
				that.getService().deleteMixEntry(that._sVersionId, oEntry.id).then(function () {
					that.showToast(that.getText("mixDeleted"));
					that._load();
				}).catch(function (oProblem) { that.showError(oProblem); });
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
