sap.ui.define([
	"sap/ui/core/mvc/Controller",
	"sap/ui/core/Fragment",
	"sap/m/MessageBox",
	"sap/m/MessageToast",
	"sap/ui/model/Sorter",
	"sugarplan/model/formatter"
], function (Controller, Fragment, MessageBox, MessageToast, Sorter, formatter) {
	"use strict";

	/**
	 * BaseController holds what every page needs: access to the service and the
	 * router, and one consistent way of reporting an error.
	 */
	return Controller.extend("sugarplan.controller.BaseController", {

		formatter: formatter,

		getService: function () {
			return this.getOwnerComponent().getService();
		},

		getRouter: function () {
			return this.getOwnerComponent().getRouter();
		},

		getAppModel: function () {
			return this.getOwnerComponent().getModel("app");
		},

		/**
		 * onContextRefresh runs fn whenever the session or the selected season
		 * changes.
		 *
		 * A page's route can match before sign-in has finished, and the season
		 * can be switched in the shell while a page is already open. Both cases
		 * mean "reload with the current context", so both raise one event and
		 * every page listens for it.
		 */
		onContextRefresh: function (fn) {
			this._fnContextRefresh = fn.bind(this);
			this.getOwnerComponent().getEventBus()
				.subscribe("app", "contextChanged", this._fnContextRefresh, this);
		},

		onExit: function () {
			if (this._fnContextRefresh) {
				this.getOwnerComponent().getEventBus()
					.unsubscribe("app", "contextChanged", this._fnContextRefresh, this);
			}
		},

		getText: function (sKey, aArgs) {
			return this.getOwnerComponent().getModel("i18n").getResourceBundle().getText(sKey, aArgs);
		},

		setBusy: function (bBusy) {
			this.getAppModel().setProperty("/busy", !!bBusy);
		},

		navTo: function (sRoute, oParams) {
			this.getRouter().navTo(sRoute, oParams);
		},

		/**
		 * showError renders a problem document.
		 *
		 * The API returns RFC 9457 problems with field-level errors addressed by
		 * row and field, so a rejected planning grid save can name the exact
		 * cell rather than saying "invalid input".
		 */
		showError: function (oProblem) {
			this.setBusy(false);

			if (!oProblem) {
				MessageBox.error(this.getText("errorUnknown"));
				return;
			}
			if (oProblem.status === 401) {
				this.getService().logout();
				this.getAppModel().setProperty("/signedIn", false);
				MessageBox.warning(this.getText("errorSessionExpired"));
				return;
			}

			var sTitle = oProblem.title || this.getText("errorTitle");
			var sDetail = oProblem.detail || "";

			if (oProblem.errors && oProblem.errors.length) {
				var aLines = oProblem.errors.slice(0, 12).map(function (oError) {
					var sWhere = oError.field || "";
					if (oError.row !== null && oError.row !== undefined) {
						sWhere = "Row " + (oError.row + 1) + (sWhere ? ", " + sWhere : "");
					}
					return (sWhere ? sWhere + ": " : "") + oError.message;
				});
				if (oProblem.errors.length > 12) {
					aLines.push("... and " + (oProblem.errors.length - 12) + " more");
				}
				sDetail = (sDetail ? sDetail + "\n\n" : "") + aLines.join("\n");
			}
			if (oProblem.correlationId) {
				sDetail += "\n\nReference: " + oProblem.correlationId;
			}

			MessageBox.error(sDetail || sTitle, { title: sTitle });
		},

		showToast: function (sMessage) {
			MessageToast.show(sMessage);
		},

		/** confirm asks before an action that is hard to undo. */
		confirm: function (sMessage, sTitle) {
			return new Promise(function (resolve) {
				MessageBox.confirm(sMessage, {
					title: sTitle,
					onClose: function (sAction) {
						resolve(sAction === MessageBox.Action.OK);
					}
				});
			});
		},

		/** requireSeason returns the selected season, sending the user to the
		 * launchpad if none has been chosen yet. */
		requireSeason: function () {
			var sSeasonId = this.getAppModel().getProperty("/selectedSeasonId");
			if (!sSeasonId) {
				this.navTo("launchpad");
				return null;
			}
			return sSeasonId;
		},

		/** can reports whether the signed-in user holds a permission. */
		/**
		 * _dialog loads a fragment once and keeps it.
		 *
		 * A dialog reloaded on every open loses its state and leaks a control
		 * tree each time, so it is cached against the controller and made a
		 * dependent of the view - which is what disposes of it when the page
		 * goes away.
		 */
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
		},

		can: function (sPermission) {
			return this.getService().can(sPermission);
		},

		// --- sorting and grouping -----------------------------------------
		//
		// Section 15 asks for sorting and grouping on the list screens. They
		// are one dialog and one binding change, so they live here rather than
		// being written out per page.
		//
		// The daily planning board is deliberately not among the pages that
		// offer them. A ledger is read in date order - the beginning balance of
		// one row is the ending balance of the row above it - and a grid sorted
		// by tonnage would show a column of continuity errors that are not
		// there. That is a case where the absence is the design, not a gap.

		/**
		 * initTableSettings wires a table to the sort and group dialog.
		 *
		 * aFields is what may be sorted or grouped by: {key, text, group}. Only
		 * fields marked group are offered for grouping, because grouping by a
		 * date or a quantity produces one group per row, which is a longer list
		 * than the one it replaced.
		 */
		initTableSettings: function (sTableId, aFields) {
			this._sSettingsTable = sTableId;
			this.getView().setModel(this.getService().newModel({
				fields: aFields,
				sortKey: aFields.length ? aFields[0].key : "",
				descending: false,
				groupKey: "",
				groupable: aFields.filter(function (o) { return o.group; })
			}), "tableSettings");
		},

		onOpenTableSettings: function () {
			this._dialog("sugarplan.view.fragment.TableSettingsDialog").then(function (oDialog) {
				oDialog.open();
			});
		},

		onCancelTableSettings: function () {
			this._closeDialog("sugarplan.view.fragment.TableSettingsDialog");
		},

		/**
		 * onApplyTableSettings re-sorts the bound list.
		 *
		 * Grouping is a sorter with the group flag rather than a separate
		 * mechanism, which is how a JSON-model list groups: the rows have to be
		 * in group order for the headers to fall in the right places, so the
		 * group sorter goes first and the chosen sort within it.
		 */
		onApplyTableSettings: function () {
			var oModel = this.getView().getModel("tableSettings");
			var oTable = this.byId(this._sSettingsTable);
			if (!oTable) {
				return;
			}
			var oBinding = oTable.getBinding("items");
			if (!oBinding) {
				return;
			}

			var sSort = oModel.getProperty("/sortKey");
			var sGroup = oModel.getProperty("/groupKey");
			var bDescending = !!oModel.getProperty("/descending");

			var aSorters = [];
			if (sGroup) {
				aSorters.push(new Sorter(sGroup, false, true));
			}
			if (sSort && sSort !== sGroup) {
				aSorters.push(new Sorter(sSort, bDescending));
			}
			oBinding.sort(aSorters);

			this._closeDialog("sugarplan.view.fragment.TableSettingsDialog");
			// The variant control cares: sort order is part of how somebody has
			// the screen set up, so a saved view that stores it must be able to
			// notice when it no longer matches.
			this.onVariantChanged();
		},

		/** tableSortState is what a page's variant collects, so a saved view
		 * restores the order as well as the filter. */
		tableSortState: function () {
			var oModel = this.getView().getModel("tableSettings");
			if (!oModel) {
				return undefined;
			}
			return {
				sortKey: oModel.getProperty("/sortKey"),
				descending: !!oModel.getProperty("/descending"),
				groupKey: oModel.getProperty("/groupKey")
			};
		},

		/** applyTableSortState puts a saved order back. */
		applyTableSortState: function (oState) {
			var oModel = this.getView().getModel("tableSettings");
			if (!oModel || !oState) {
				return;
			}
			oModel.setProperty("/sortKey", oState.sortKey || "");
			oModel.setProperty("/descending", !!oState.descending);
			oModel.setProperty("/groupKey", oState.groupKey || "");
			this.onApplyTableSettings();
		},

		// --- variant management -------------------------------------------
		//
		// Section 15 asks for variant management, saved views and
		// personalization. Stored, all three are the same thing: what a page
		// looked like when somebody had it the way they wanted, under a name
		// they can find again.
		//
		// The behaviour lives here rather than in each page because the parts
		// that are easy to get subtly wrong - applying a default exactly once,
		// not offering delete on somebody else's variant, keeping the
		// "modified" marker honest - should be got right once. A page supplies
		// two functions and gets the rest:
		//
		//     this.initVariants("board", {
		//         collect: function () { ... return the current filter; },
		//         apply:   function (oPayload) { ... put the page into it; }
		//     });

		/**
		 * initVariants wires this page to its saved views.
		 *
		 * sPage is the route name, which is also what the server files the view
		 * under, so a variant can never be offered on a screen that could not
		 * apply it.
		 */
		initVariants: function (sPage, oHandlers, bSkipDefault) {
			this._sVariantPage = sPage;
			this._oVariantHandlers = oHandlers;
			// A page arrived at through a drill-down carries the filter that was
			// clicked, and a saved default must not overwrite it: somebody who
			// clicked through to a period meant that period.
			this._bVariantDefaultApplied = !!bSkipDefault;
			this.getView().setModel(this.getService().newModel({
				items: [], selectedId: "", selectedName: "", modified: false,
				owned: false, canShare: false, name: "", shared: false, isDefault: false
			}), "variants");
			return this.reloadVariants();
		},

		/**
		 * reloadVariants fetches the list and applies the default the first time.
		 *
		 * The default is applied once per page visit, not on every refresh: a
		 * planner who has narrowed the dates and then pressed refresh means
		 * "show me that again", not "throw away what I just set up".
		 */
		reloadVariants: function () {
			var that = this;
			return this.getService().listViews(this._sVariantPage).then(function (oPage) {
				var aItems = oPage.value || [];
				var oModel = that.getView().getModel("variants");
				oModel.setProperty("/items", aItems);
				// Sharing needs an account scoped to exactly one factory,
				// because that is the factory a shared view is published to.
				var oProfile = that.getService().getSessionProfile() || {};
				oModel.setProperty("/canShare", (oProfile.factories || []).length === 1);

				if (that._bVariantDefaultApplied) {
					that._refreshVariantSelection();
					return aItems;
				}
				that._bVariantDefaultApplied = true;
				var oDefault = aItems.filter(function (oItem) { return oItem.isDefault; })[0];
				if (oDefault) {
					that._selectVariant(oDefault);
				}
				return aItems;
			});
		},

		/** onVariantSelect applies the variant the user picked. */
		onVariantSelect: function (oEvent) {
			var sId = oEvent.getSource().getSelectedKey();
			var aItems = this.getView().getModel("variants").getProperty("/items") || [];
			var oItem = aItems.filter(function (o) { return o.id === sId; })[0];
			if (!oItem) {
				// The empty entry is "no variant": the page keeps whatever the
				// user has set rather than being reset, because clearing a
				// selection is not the same as asking for the defaults back.
				this.getView().getModel("variants").setProperty("/selectedId", "");
				this._markVariantModified();
				return;
			}
			this._selectVariant(oItem);
		},

		_selectVariant: function (oItem) {
			var oModel = this.getView().getModel("variants");
			oModel.setProperty("/selectedId", oItem.id);
			oModel.setProperty("/selectedName", oItem.name);
			this._sVariantApplied = JSON.stringify(oItem.payload);
			this._refreshVariantSelection();
			this._oVariantHandlers.apply(oItem.payload);
		},

		/** _refreshVariantSelection keeps the buttons honest about what may be
		 * done to the selected variant. */
		_refreshVariantSelection: function () {
			var oModel = this.getView().getModel("variants");
			var sId = oModel.getProperty("/selectedId");
			var oItem = (oModel.getProperty("/items") || []).filter(function (o) {
				return o.id === sId;
			})[0];
			var oProfile = this.getService().getSessionProfile() || {};
			// Only the owner may change or delete a variant; a shared one that
			// anybody could edit is one nobody could rely on.
			oModel.setProperty("/owned", !!oItem && oItem.owner === oProfile.username);
			oModel.setProperty("/modified", false);
		},

		/**
		 * onVariantChanged is called by a page when its filter moves, so the
		 * control can say the variant no longer matches what is on screen.
		 */
		onVariantChanged: function () {
			this._markVariantModified();
		},

		_markVariantModified: function () {
			var oModel = this.getView().getModel("variants");
			if (!oModel || !this._oVariantHandlers) {
				return;
			}
			// "Modified" means what is on screen differs from the variant, not
			// that an event fired: a page that re-applies the same filter has
			// not modified anything, and a marker that cried wolf would be
			// ignored.
			var sNow = JSON.stringify(this._oVariantHandlers.collect());
			oModel.setProperty("/modified",
				!!oModel.getProperty("/selectedId") && sNow !== this._sVariantApplied);
		},

		/**
		 * onVariantSaveAs opens the dialog to store the current filter.
		 *
		 * The dialog is seeded from the selected variant, when there is one you
		 * own. Saving replaces the whole record, so a dialog that opened with
		 * the boxes clear would quietly un-default and un-share a view every
		 * time its owner adjusted the dates and pressed save.
		 */
		onVariantSaveAs: function () {
			var oModel = this.getView().getModel("variants");
			var sId = oModel.getProperty("/selectedId");
			var oSelected = (oModel.getProperty("/items") || []).filter(function (o) {
				return o.id === sId;
			})[0];
			var oProfile = this.getService().getSessionProfile() || {};
			var bMine = !!oSelected && oSelected.owner === oProfile.username;

			oModel.setProperty("/name", bMine ? oSelected.name : "");
			oModel.setProperty("/shared", bMine && !!oSelected.shared);
			oModel.setProperty("/isDefault", bMine && !!oSelected.isDefault);
			this._dialog("sugarplan.view.fragment.VariantDialog").then(function (oDialog) {
				oDialog.open();
			});
		},

		onVariantCancel: function () {
			this._closeDialog("sugarplan.view.fragment.VariantDialog");
		},

		onVariantSave: function () {
			var oModel = this.getView().getModel("variants");
			var sName = (oModel.getProperty("/name") || "").trim();
			if (!sName) {
				this.showToast(this.getText("variantNameRequired"));
				return;
			}
			var that = this;
			this.setBusy(true);
			this.getService().saveView({
				page: this._sVariantPage,
				name: sName,
				shared: !!oModel.getProperty("/shared"),
				isDefault: !!oModel.getProperty("/isDefault"),
				payload: this._oVariantHandlers.collect()
			}).then(function (oSaved) {
				that.setBusy(false);
				that._closeDialog("sugarplan.view.fragment.VariantDialog");
				that.showToast(that.getText("variantSaved", [oSaved.name]));
				return that.reloadVariants().then(function () {
					that._selectVariant(oSaved);
				});
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		onVariantDelete: function () {
			var oModel = this.getView().getModel("variants");
			var sId = oModel.getProperty("/selectedId");
			var sName = oModel.getProperty("/selectedName");
			if (!sId) {
				return;
			}
			var that = this;
			this.confirm(this.getText("variantDeleteConfirm", [sName]),
				this.getText("variantDeleteTitle")).then(function (bYes) {
				if (!bYes) {
					return;
				}
				that.setBusy(true);
				that.getService().deleteView(sId).then(function () {
					that.setBusy(false);
					oModel.setProperty("/selectedId", "");
					oModel.setProperty("/selectedName", "");
					that.showToast(that.getText("variantDeleted", [sName]));
					return that.reloadVariants();
				}).catch(function (oProblem) {
					that.showError(oProblem);
				});
			});
		},

		/** onVariantSetDefault makes the selected variant the one this page
		 * opens with. */
		onVariantSetDefault: function () {
			var oModel = this.getView().getModel("variants");
			var sId = oModel.getProperty("/selectedId");
			if (!sId) {
				return;
			}
			var oItem = (oModel.getProperty("/items") || []).filter(function (o) {
				return o.id === sId;
			})[0];
			var that = this;
			this.setBusy(true);
			this.getService().setDefaultView(sId, !(oItem && oItem.isDefault))
				.then(function () {
					that.setBusy(false);
					that.showToast(that.getText("variantDefaultSet"));
					return that.reloadVariants();
				}).catch(function (oProblem) {
					that.showError(oProblem);
				});
		}
	});
});
