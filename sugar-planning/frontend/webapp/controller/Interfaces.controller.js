sap.ui.define([
	"sugarplan/controller/BaseController"
], function (BaseController) {
	"use strict";

	/**
	 * Interfaces is the operations view of what this system sends out.
	 *
	 * It answers the question somebody asks after an outage: did the ERP hear
	 * about the shipments we posted this morning? The outbox is the honest
	 * answer, including the events that have given up and are waiting for a
	 * person - which is the list that matters and the one that is easiest to
	 * leave off a screen.
	 */
	return BaseController.extend("sugarplan.controller.Interfaces", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({
				events: [],
				topic: "",
				scope: "unpublished",
				canRead: false,
				waiting: 0,
				stuck: 0,
				published: 0
			}), "view");
			this.getRouter().getRoute("interfaces").attachPatternMatched(this._onDisplay, this);
			this.onContextRefresh(this._onDisplay);
		},

		_onDisplay: function () {
			var bCanRead = this.can("integration:read");
			this.getView().getModel("view").setProperty("/canRead", bCanRead);
			if (!bCanRead) {
				this.getView().getModel("view").setProperty("/events", []);
				return;
			}
			this._load();
		},

		_load: function () {
			var oModel = this.getView().getModel("view");
			var sScope = oModel.getProperty("/scope");
			var that = this;
			this.setBusy(true);

			// The counts come from three narrow reads rather than one wide one,
			// so a season's worth of published events never has to cross the
			// wire just to show that none of them are stuck.
			var oService = this.getService();
			var oParams = { topic: oModel.getProperty("/topic") };
			if (sScope === "unpublished") {
				oParams.unpublished = "true";
			} else if (sScope === "exhausted") {
				oParams.exhausted = "true";
			}

			Promise.all([
				oService.listEvents(oParams),
				oService.listEvents({ unpublished: "true", $top: 1 }),
				oService.listEvents({ exhausted: "true", $top: 1 }),
				oService.listEvents({ $top: 1 })
			]).then(function (aResults) {
				oModel.setProperty("/events", aResults[0].value || []);
				var iWaiting = aResults[1].count || 0;
				var iStuck = aResults[2].count || 0;
				var iAll = aResults[3].count || 0;
				oModel.setProperty("/waiting", iWaiting);
				oModel.setProperty("/stuck", iStuck);
				oModel.setProperty("/published", Math.max(iAll - iWaiting, 0));
				that.setBusy(false);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		onFilterChange: function () {
			this._load();
		},

		onRefresh: function () {
			this._load();
		},

		/** onDispatch drains what is due now, which is what somebody does once
		 * the far end is back rather than waiting for the next tick. */
		onDispatch: function () {
			var that = this;
			this.setBusy(true);
			this.getService().dispatch().then(function (oResult) {
				that.setBusy(false);
				that.showToast(that.getText("dispatchDone", [
					oResult.published || 0, oResult.failed || 0
				]));
				that._load();
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		/** onRetry delivers one event now, ignoring its backoff. */
		onRetry: function (oEvent) {
			var oItem = oEvent.getSource().getBindingContext("view").getObject();
			var that = this;
			this.setBusy(true);
			this.getService().retryEvent(oItem.id).then(function () {
				that.setBusy(false);
				that.showToast(that.getText("retryDelivered"));
				that._load();
			}).catch(function (oProblem) {
				// A far end that is still broken is not this operator's mistake,
				// so the message is the one the far end gave.
				that.showError(oProblem);
				that._load();
			});
		},

		onNavBack: function () {
			this.navTo("launchpad");
		}
	});
});
