sap.ui.define([
	"sugarplan/controller/BaseController"
], function (BaseController) {
	"use strict";

	/**
	 * Orders is the shift supervisor's page: what the floor has been told to
	 * make, and what it actually made.
	 *
	 * Confirming is the one action here that moves stock, and it does so through
	 * the same posting rules as anything else: the yield is receipted into a
	 * store, and if that store is full the confirmation is refused rather than
	 * leaving a confirmed order whose sugar is nowhere.
	 */
	return BaseController.extend("sugarplan.controller.Orders", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({
				orders: [],
				warehouses: [],
				products: [],
				openOnly: true,
				canProduce: false,
				canApprove: false,
				detail: null,
				confirm: this._emptyConfirm(),
				fromPlan: { from: "", to: "" },
				releasedVersionId: "",
				releasedVersionCode: ""
			}), "view");
			this.getRouter().getRoute("orders").attachPatternMatched(this._onDisplay, this);
			this.onContextRefresh(this._onDisplay);
		},

		_emptyConfirm: function () {
			return {
				businessDate: new Date().toISOString().slice(0, 10),
				yieldQty: "",
				scrapQty: "",
				reworkQty: "",
				warehouseId: ""
			};
		},

		_onDisplay: function () {
			if (!this.getAppModel().getProperty("/signedIn")) {
				return;
			}
			var oModel = this.getView().getModel("view");
			var oService = this.getService();
			var that = this;

			oModel.setProperty("/canProduce", this.can("actual:production"));
			oModel.setProperty("/canApprove", this.can("plan:approve"));
			this.setBusy(true);

			var sSeasonId = this.getAppModel().getProperty("/selectedSeasonId");
			Promise.all([
				oService.listOrders({ openOnly: oModel.getProperty("/openOnly") || undefined, $top: 200 }),
				oService.listMaster("warehouses", { active: "true" }),
				sSeasonId ? oService.listVersions(sSeasonId) : Promise.resolve({ value: [] }),
				oService.listMaster("products", { active: "true" })
			]).then(function (aResults) {
				// An order carries the product's id, which is meaningless on a
				// screen. The master data is loaded once and the label put on
				// each row, rather than asking the API for a denormalised list.
				var mProducts = {};
				(aResults[3].value || []).forEach(function (oProduct) {
					mProducts[oProduct.id] = oProduct;
				});
				var aOrders = (aResults[0].value || []).map(function (oOrder) {
					var oProduct = mProducts[oOrder.productId];
					oOrder.productCode = oProduct ? oProduct.code : "";
					oOrder.productName = oProduct ? oProduct.name : oOrder.productId;
					return oOrder;
				});
				oModel.setProperty("/orders", aOrders);
				oModel.setProperty("/products", aResults[3].value || []);
				oModel.setProperty("/warehouses", aResults[1].value || []);

				// Orders may only be raised from a released plan, so the page
				// finds it once and disables the button when there is none.
				//
				// The actuals container is released from the day it is created,
				// so operators can post to it, but it is not a plan: it records
				// what happened rather than what to make. Skipping it here is
				// what stops the button offering an empty run.
				var oReleased = (aResults[2].value || []).filter(function (oVersion) {
					return oVersion.status === "RELEASED" && oVersion.planType !== "ACTUAL";
				})[0];
				oModel.setProperty("/releasedVersionId", oReleased ? oReleased.id : "");
				oModel.setProperty("/releasedVersionCode", oReleased ? oReleased.code : "");
				that.setBusy(false);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		onToggleOpenOnly: function (oEvent) {
			this.getView().getModel("view").setProperty("/openOnly", oEvent.getParameter("selected"));
			this._onDisplay();
		},

		// ------------------------------------------------------------------
		// Creating orders from the released plan
		// ------------------------------------------------------------------

		onOpenFromPlan: function () {
			var sVersionId = this.getView().getModel("view").getProperty("/releasedVersionId");
			if (!sVersionId) {
				this.showError({
					title: this.getText("ordersNoReleasedTitle"),
					detail: this.getText("ordersNoReleased")
				});
				return;
			}
			this._dialog("sugarplan.view.fragment.OrdersFromPlanDialog").then(function (oDialog) {
				oDialog.open();
			});
		},

		onCancelFromPlan: function () {
			this._closeDialog("sugarplan.view.fragment.OrdersFromPlanDialog");
		},

		onSubmitFromPlan: function () {
			var oModel = this.getView().getModel("view");
			var oRange = oModel.getProperty("/fromPlan");
			var sVersionId = oModel.getProperty("/releasedVersionId");
			var that = this;

			if (!oRange.from || !oRange.to) {
				this.showError({
					title: this.getText("ordersRangeTitle"),
					detail: this.getText("ordersRangeRequired")
				});
				return;
			}

			this.setBusy(true);
			this.getService().ordersFromPlan(sVersionId, {
				from: oRange.from, to: oRange.to
			}).then(function (oResult) {
				that._closeDialog("sugarplan.view.fragment.OrdersFromPlanDialog");
				var iCreated = (oResult.created || []).length;
				var iSkipped = (oResult.skipped || []).length;
				// Both numbers are reported: "nothing created" is a normal and
				// useful answer when the range is already covered.
				that.showToast(that.getText("ordersCreated", [iCreated, iSkipped]));
				that._onDisplay();
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		// ------------------------------------------------------------------
		// Order actions
		// ------------------------------------------------------------------

		onOpenOrder: function (oEvent) {
			var oItem = oEvent.getParameter("listItem") || oEvent.getSource();
			var oContext = oItem.getBindingContext("view");
			if (!oContext) {
				return;
			}
			this._loadDetail(oContext.getObject().id);
		},

		_loadDetail: function (sOrderId) {
			var that = this;
			this.setBusy(true);
			return this.getService().getOrder(sOrderId).then(function (oDetail) {
				that.getView().getModel("view").setProperty("/detail", oDetail);
				that.setBusy(false);
				return that._dialog("sugarplan.view.fragment.OrderDetailDialog");
			}).then(function (oDialog) {
				oDialog.open();
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		onCloseDetail: function () {
			this._closeDialog("sugarplan.view.fragment.OrderDetailDialog");
		},

		onRelease: function () {
			this._act("RELEASE");
		},

		onCancelOrder: function () {
			var that = this;
			this.confirm(this.getText("ordersCancelConfirm"), this.getText("ordersCancelTitle"))
				.then(function (bOk) {
					if (bOk) {
						that._act("CANCEL");
					}
				});
		},

		/** onTechnicallyClose closes an order whose quantity is reconciled. An
		 * order outside the tolerance needs a reason code and an approver, and
		 * the API says so rather than this page guessing. */
		onTechnicallyClose: function () {
			this._act("TECHNICALLY_CLOSE", {
				varianceReason: this.getView().getModel("view").getProperty("/detail/order/varianceReason")
			});
		},

		_act: function (sAction, oExtra) {
			var oModel = this.getView().getModel("view");
			var oOrder = oModel.getProperty("/detail/order");
			var that = this;

			this.setBusy(true);
			this.getService().orderAction(oOrder.id,
				Object.assign({ action: sAction, rowVersion: oOrder.rowVersion }, oExtra || {})
			).then(function () {
				that.showToast(that.getText("ordersActionDone", [sAction]));
				return that.getService().getOrder(oOrder.id);
			}).then(function (oDetail) {
				oModel.setProperty("/detail", oDetail);
				that.setBusy(false);
				that._onDisplay();
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		// ------------------------------------------------------------------
		// Confirmation
		// ------------------------------------------------------------------

		onOpenConfirm: function () {
			var oModel = this.getView().getModel("view");
			var oOrder = oModel.getProperty("/detail/order");
			var oConfirm = this._emptyConfirm();
			// The open quantity is offered as the default: a shift that made
			// what it was asked to make presses one button.
			oConfirm.businessDate = oOrder.businessDate;
			oConfirm.yieldQty = oModel.getProperty("/detail/openQty");
			oModel.setProperty("/confirm", oConfirm);

			this._dialog("sugarplan.view.fragment.ConfirmDialog").then(function (oDialog) {
				oDialog.open();
			});
		},

		onCancelConfirm: function () {
			this._closeDialog("sugarplan.view.fragment.ConfirmDialog");
		},

		onSubmitConfirm: function () {
			var oModel = this.getView().getModel("view");
			var oOrder = oModel.getProperty("/detail/order");
			var oConfirm = oModel.getProperty("/confirm");
			var that = this;

			this.setBusy(true);
			this.getService().confirmOrder(oOrder.id, {
				businessDate: oConfirm.businessDate,
				yieldQty: String(oConfirm.yieldQty || "0"),
				scrapQty: String(oConfirm.scrapQty || "0"),
				reworkQty: String(oConfirm.reworkQty || "0"),
				warehouseId: oConfirm.warehouseId || undefined
			}).then(function (oResult) {
				that._closeDialog("sugarplan.view.fragment.ConfirmDialog");
				that.showToast(that.getText("ordersConfirmed",
					[oResult.confirmation.confirmationNo]));
				return that.getService().getOrder(oOrder.id);
			}).then(function (oDetail) {
				oModel.setProperty("/detail", oDetail);
				that._onDisplay();
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		/** onReverseConfirmation undoes a confirmation: the goods receipt is
		 * reversed and the order's confirmed quantity comes back down. */
		onReverseConfirmation: function (oEvent) {
			var oConfirmation = oEvent.getSource().getBindingContext("view").getObject();
			var oModel = this.getView().getModel("view");
			var sOrderId = oModel.getProperty("/detail/order/id");
			var that = this;

			this.confirm(this.getText("ordersReverseConfirm", [oConfirmation.confirmationNo]),
				this.getText("ordersReverseTitle")).then(function (bOk) {
				if (!bOk) {
					return;
				}
				that.setBusy(true);
				that.getService().reverseConfirmation(oConfirmation.id, {
					reason: that.getText("ordersReverseReason")
				}).then(function () {
					return that.getService().getOrder(sOrderId);
				}).then(function (oDetail) {
					oModel.setProperty("/detail", oDetail);
					that.showToast(that.getText("ordersReversed"));
					that._onDisplay();
				}).catch(function (oProblem) {
					that.showError(oProblem);
				});
			});
		},

		/** allowed reports whether an action is legal in the order's current
		 * status, so a button that would be refused is disabled instead. */
		isAllowed: function (aActions, sAction) {
			return !!(aActions && aActions.indexOf(sAction) >= 0);
		},

		onRefresh: function () {
			this._onDisplay();
		},

		onNavBack: function () {
			this.navTo("launchpad");
		}
	});
});
