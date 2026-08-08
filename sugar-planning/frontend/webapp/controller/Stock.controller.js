sap.ui.define([
	"sugarplan/controller/BaseController"
], function (BaseController) {
	"use strict";

	/**
	 * Stock is the warehouse keeper's page: what is in each store now, what is
	 * blocked, and the movements that got it there.
	 *
	 * Everything on this page goes through the posting endpoint, so the rules
	 * about negative stock, held stock and capacity are the same ones the API
	 * enforces. The page never edits a balance; it posts a document and reads
	 * the balance back.
	 */
	return BaseController.extend("sugarplan.controller.Stock", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({
				stock: [],
				documents: [],
				warehouses: [],
				products: [],
				canPost: false,
				posting: this._emptyPosting()
			}), "view");
			this.getRouter().getRoute("stock").attachPatternMatched(this._onDisplay, this);
			this.onContextRefresh(this._onDisplay);
		},

		_emptyPosting: function () {
			return {
				docType: "RECEIPT",
				businessDate: new Date().toISOString().slice(0, 10),
				warehouseId: "",
				toWarehouse: "",
				productId: "",
				quantity: "",
				note: "",
				capacityOverride: false
			};
		},

		_onDisplay: function () {
			if (!this.getAppModel().getProperty("/signedIn")) {
				return;
			}
			var oModel = this.getView().getModel("view");
			var oService = this.getService();
			var that = this;

			oModel.setProperty("/canPost", this.can("actual:stock"));
			this.setBusy(true);

			Promise.all([
				oService.listStock({}),
				oService.listDocuments({ factoryId: this._factoryId(), $top: 100 }),
				oService.listMaster("warehouses", { active: "true" }),
				oService.listMaster("products", { active: "true" })
			]).then(function (aResults) {
				oModel.setProperty("/stock", aResults[0].value || []);
				oModel.setProperty("/documents", aResults[1].value || []);
				oModel.setProperty("/warehouses", aResults[2].value || []);
				oModel.setProperty("/products", aResults[3].value || []);
				that.setBusy(false);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		_factoryId: function () {
			var oProfile = this.getService().getSessionProfile() || {};
			var aFactories = oProfile.factories || [];
			return aFactories.length === 1 ? aFactories[0] : "";
		},

		// ------------------------------------------------------------------
		// Posting
		// ------------------------------------------------------------------

		onOpenPosting: function () {
			var oModel = this.getView().getModel("view");
			oModel.setProperty("/posting", this._emptyPosting());
			this._dialog("sugarplan.view.fragment.PostingDialog").then(function (oDialog) {
				oDialog.open();
			});
		},

		/** onPostingTypeChange clears the receiving store when the movement is
		 * no longer a transfer, so a stale value cannot be submitted. */
		onPostingTypeChange: function () {
			var oModel = this.getView().getModel("view");
			if (oModel.getProperty("/posting/docType") !== "TRANSFER") {
				oModel.setProperty("/posting/toWarehouse", "");
			}
		},

		onCancelPosting: function () {
			this._closeDialog("sugarplan.view.fragment.PostingDialog");
		},

		onSubmitPosting: function () {
			var oModel = this.getView().getModel("view");
			var oPosting = oModel.getProperty("/posting");
			var that = this;

			if (!oPosting.warehouseId || !oPosting.productId || !oPosting.quantity) {
				this.showError({
					title: this.getText("stockPostingIncompleteTitle"),
					detail: this.getText("stockPostingIncomplete")
				});
				return;
			}

			var oRequest = {
				docType: oPosting.docType,
				businessDate: oPosting.businessDate,
				factoryId: this._factoryId() || this._warehouseFactory(oPosting.warehouseId),
				note: oPosting.note,
				capacityOverride: !!oPosting.capacityOverride,
				lines: [{
					warehouseId: oPosting.warehouseId,
					toWarehouse: oPosting.docType === "TRANSFER" ? oPosting.toWarehouse : undefined,
					productId: oPosting.productId,
					quantity: String(oPosting.quantity)
				}]
			};

			this.setBusy(true);
			this.getService().postDocument(oRequest).then(function (oDocument) {
				that._closeDialog("sugarplan.view.fragment.PostingDialog");
				that.showToast(that.getText("stockPosted", [oDocument.documentNo]));
				that._onDisplay();
			}).catch(function (oProblem) {
				// The dialog stays open: the keeper corrects the line rather
				// than typing the whole movement again.
				that.showError(oProblem);
			});
		},

		_warehouseFactory: function (sWarehouseId) {
			var aWarehouses = this.getView().getModel("view").getProperty("/warehouses") || [];
			for (var i = 0; i < aWarehouses.length; i++) {
				if (aWarehouses[i].id === sWarehouseId) {
					return aWarehouses[i].factoryId;
				}
			}
			return "";
		},

		/** onReverseDocument undoes a posting. Nothing is deleted: the counter
		 * document is posted and both stay in the ledger. */
		onReverseDocument: function (oEvent) {
			var oDocument = oEvent.getSource().getBindingContext("view").getObject();
			var that = this;

			if (oDocument.reversed) {
				this.showToast(this.getText("stockAlreadyReversed"));
				return;
			}
			this.confirm(this.getText("stockReverseConfirm", [oDocument.documentNo]),
				this.getText("stockReverseTitle")).then(function (bOk) {
				if (!bOk) {
					return;
				}
				that.setBusy(true);
				that.getService().reverseDocument(oDocument.id, {
					reason: that.getText("stockReverseReason")
				}).then(function (oReversal) {
					that.showToast(that.getText("stockReversed", [oReversal.documentNo]));
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
