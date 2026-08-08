sap.ui.define([
	"sap/ui/core/mvc/Controller",
	"sap/ui/core/Fragment",
	"sap/ui/model/json/JSONModel",
	"sap/m/Button",
	"sap/m/Dialog",
	"sap/m/MessageBox",
	"sap/m/MessageToast",
	"sap/m/Text",
	"sap/m/TextArea",
	"farm/area/dashboard/model/formatter"
], function (Controller, Fragment, JSONModel, Button, Dialog, MessageBox, MessageToast, Text, TextArea, formatter) {
	"use strict";

	// How each action is offered. The server decides whether an action is possible; this only says
	// what it looks like and whether it is worth a confirmation.
	var ACTION_STYLE = {
		Submit: { type: "Emphasized", icon: "sap-icon://outbox" },
		Review: { type: "Default", icon: "sap-icon://inspection" },
		Approve: { type: "Accept", icon: "sap-icon://accept", confirm: true },
		Reject: { type: "Reject", icon: "sap-icon://decline", needsReason: true },
		ReturnForCorrection: { type: "Default", icon: "sap-icon://undo", needsReason: true },
		// Revise asks for a reason rather than a confirmation — the reason prompt says what
		// revising does, so a separate "are you sure" would be a second click for nothing.
		Revise: { type: "Default", icon: "sap-icon://copy", needsReason: true },
		Close: { type: "Default", icon: "sap-icon://locked", confirm: true }
	};

	var LABEL = {
		Submit: "Submit",
		Review: "Start review",
		Approve: "Approve",
		Reject: "Reject",
		ReturnForCorrection: "Return for correction",
		Revise: "Open a revision",
		Close: "Close"
	};

	return Controller.extend("farm.area.dashboard.controller.Projection", {
		formatter: formatter,

		onInit: function () {
			var view = this.getView();
			view.addStyleClass(this.getOwnerComponent().getContentDensityClass());

			view.setModel(new JSONModel({ lines: [], history: [], actions: [] }), "p");
			view.setModel(new JSONModel({
				seedCaneTons: 0, workflowHint: "", hasChain: false, chainText: "", chainLink: "",
				selectedBlockArea: ""
			}), "view");
			view.setModel(new JSONModel({ blocksOnly: [], varietiesOnly: [] }), "lookups");

			this.getOwnerComponent().getRouter().getRoute("projection")
				.attachPatternMatched(this._onRouteMatched, this);
		},

		_api: function () {
			return this.getOwnerComponent().getApi();
		},

		_onRouteMatched: function (event) {
			if (!this._api().isSignedIn()) {
				this.getOwnerComponent().getRouter().navTo("login");
				return;
			}
			this.getOwnerComponent().getModel("session").setProperty("/user", this._api().currentUser());
			this._id = event.getParameter("arguments").id;

			if (!this._loaded) {
				this._loaded = true;
				this._loadLookups();
			}
			this._reload();
		},

		_loadLookups: function () {
			var model = this.getView().getModel("lookups");
			this._api().get("lookups/blocks").then(function (blocks) {
				model.setProperty("/blocksOnly", blocks || []);
			}).catch(this._showError.bind(this));
			this._api().get("lookups/varieties").then(function (varieties) {
				model.setProperty("/varietiesOnly", varieties || []);
			}).catch(this._showError.bind(this));
			// The plantable area of every block, so the line dialog can show the ceiling before the
			// planner types rather than after the server refuses.
			this._api().get("blocks", { pageSize: 500 }).then(function (page) {
				this._blockAreas = {};
				((page && page.items) || []).forEach(function (b) {
					this._blockAreas[b.id] = b.plantableAreaHa;
				}.bind(this));
			}.bind(this)).catch(function () { /* the server still enforces the ceiling */ });
		},

		_reload: function () {
			this.byId("errorStrip").setVisible(false);
			return this._api().get("projections/" + this._id).then(function (projection) {
				this._apply(projection);
			}.bind(this)).catch(this._showError.bind(this));
		},

		_apply: function (projection) {
			this.getView().getModel("p").setData(projection);

			var lines = projection.lines || [];
			var seedCane = lines.reduce(function (sum, l) { return sum + (l.requiredSeedCaneTons || 0); }, 0);

			var chain = "";
			var link = "";
			if (projection.supersedesId) {
				chain = "This is version " + projection.revision + ". It replaced an earlier plan.";
				link = "Open version " + (projection.revision - 1);
				this._chainTarget = projection.supersedesId;
			} else if (projection.supersededById) {
				chain = "This version has been superseded by a later one.";
				link = "Open the current version";
				this._chainTarget = projection.supersededById;
			}

			this.getView().getModel("view").setData({
				seedCaneTons: seedCane,
				workflowHint: this._hintFor(projection),
				hasChain: !!chain,
				chainText: chain,
				chainLink: link,
				selectedBlockArea: ""
			});

			this._buildWorkflowButtons(projection.actions || []);
		},

		_hintFor: function (projection) {
			if (projection.linesEditable) {
				return projection.lineCount
					? "A draft. Submit it when the blocks are right."
					: "A draft with no blocks yet. Add at least one before submitting.";
			}
			if (!(projection.actions || []).length) {
				return "Nothing for you to do on this plan.";
			}
			return "";
		},

		// The footer is built from the actions the server returned for this caller, not from a copy
		// of the workflow held here. A button that appears always works.
		_buildWorkflowButtons: function (actions) {
			var bar = this.byId("workflowBar");
			(this._actionButtons || []).forEach(function (button) {
				bar.removeContent(button);
				button.destroy();
			});
			this._actionButtons = actions.map(function (action) {
				var style = ACTION_STYLE[action] || {};
				var button = new Button({
					text: LABEL[action] || action,
					type: style.type || "Default",
					icon: style.icon,
					press: this._onAction.bind(this, action)
				});
				bar.addContent(button);
				return button;
			}.bind(this));
		},

		_onAction: function (action) {
			var style = ACTION_STYLE[action] || {};
			if (style.needsReason) {
				this._askForReason(action);
				return;
			}
			if (style.confirm) {
				MessageBox.confirm(confirmationFor(action), {
					title: LABEL[action] || action,
					onClose: function (choice) {
						if (choice === MessageBox.Action.OK) {
							this._send(action, "");
						}
					}.bind(this)
				});
				return;
			}
			this._send(action, "");
		},

		// Rejecting, returning and revising all need a reason — the server refuses them without
		// one, so asking here saves a round trip and puts the prompt where the decision is made.
		// MessageBox cannot host an input, so this is a small dialog of its own, built once.
		_askForReason: function (action) {
			if (!this._reasonInput) {
				this._reasonInput = new TextArea({ rows: 3, width: "100%", placeholder: "Reason" });
				this._reasonDialog = new Dialog({
					contentWidth: "30rem",
					content: [new Text({ text: "" }).addStyleClass("sapUiSmallMargin"), this._reasonInput],
					beginButton: new Button({
						text: "Confirm",
						type: "Emphasized",
						press: function () {
							var reason = this._reasonInput.getValue().trim();
							if (!reason) {
								this._reasonInput.setValueState("Error");
								this._reasonInput.setValueStateText("A reason is required.");
								return;
							}
							this._reasonDialog.close();
							this._send(this._reasonAction, reason);
						}.bind(this)
					}),
					endButton: new Button({
						text: "Cancel",
						press: function () { this._reasonDialog.close(); }.bind(this)
					})
				});
				this.getView().addDependent(this._reasonDialog);
			}

			this._reasonAction = action;
			this._reasonInput.setValue("");
			this._reasonInput.setValueState("None");
			this._reasonDialog.setTitle(LABEL[action] || action);
			this._reasonDialog.getContent()[0].setText(reasonPromptFor(action));
			this._reasonDialog.open();
		},

		_send: function (action, comments) {
			var version = this.getView().getModel("p").getProperty("/version");
			this.byId("errorStrip").setVisible(false);
			this._api().post("projections/" + this._id + "/" + action.toLowerCase(), {
				comments: comments || "",
				version: version
			}).then(function (projection) {
				MessageToast.show((LABEL[action] || action) + " — the plan is now " + projection.status + ".");
				// Revising answers with the new draft, which is a different plan; follow it.
				if (projection.id !== Number(this._id)) {
					this.getOwnerComponent().getRouter().navTo("projection", { id: projection.id });
					return;
				}
				this._apply(projection);
			}.bind(this)).catch(this._showError.bind(this));
		},

		onOpenChain: function () {
			this.getOwnerComponent().getRouter().navTo("projection", { id: this._chainTarget });
		},

		// ---------------------------------------------------------------- lines

		onAddLine: function () {
			this._editingLineId = null;
			this._lineDialog().then(function (dialog) {
				var header = this.getView().getModel("p").getData();
				this.byId("lineDialogError").setVisible(false);
				this.byId("dlgBlock").setSelectedKey("");
				this.byId("dlgBlock").setValue("");
				// The variety is required, so it starts on the estate's first one rather than
				// blank: a required dropdown with nothing in it is a trap, not a choice.
				var varieties = this.getView().getModel("lookups").getProperty("/varietiesOnly") || [];
				this.byId("dlgVariety").setSelectedKey(varieties.length ? String(varieties[0].id) : "");
				this.byId("dlgPlantingType").setSelectedKey("NewPlanting");
				this.byId("dlgArea").setValue("");
				this.byId("dlgLineStart").setValue(header.planningStart || "");
				this.byId("dlgLineEnd").setValue(header.planningEnd || "");
				this.byId("dlgHarvest").setValue("");
				this.byId("dlgPriority").setSelectedKey("5");
				this.byId("dlgLineRemark").setValue("");
				this.getView().getModel("view").setProperty("/selectedBlockArea", "");
				dialog.open();
			}.bind(this));
		},

		onEditLine: function (event) {
			var line = event.getSource().getBindingContext("p").getObject();
			this._editingLineId = line.id;
			this._lineDialog().then(function (dialog) {
				this.byId("lineDialogError").setVisible(false);
				this.byId("dlgBlock").setSelectedKey(String(line.blockId));
				this.byId("dlgVariety").setSelectedKey(String(line.caneVarietyId));
				this.byId("dlgPlantingType").setSelectedKey(line.plantingType);
				this.byId("dlgArea").setValue(String(line.projectedPlantingAreaHa));
				this.byId("dlgLineStart").setValue(line.plannedPlantingStart);
				this.byId("dlgLineEnd").setValue(line.plannedPlantingEnd);
				this.byId("dlgHarvest").setValue(line.expectedHarvestDate || "");
				this.byId("dlgPriority").setSelectedKey(String(line.priority));
				this.byId("dlgLineRemark").setValue(line.remark || "");
				this.getView().getModel("view").setProperty("/selectedBlockArea",
					formatter.hectares(line.availableAreaHa) + " ha");
				dialog.open();
			}.bind(this));
		},

		onLineBlockChanged: function () {
			var id = this._chosenBlockId();
			var area = this._blockAreas && this._blockAreas[id];
			this.getView().getModel("view").setProperty("/selectedBlockArea",
				typeof area === "number" ? formatter.hectares(area) + " ha" : "");
		},

		// A ComboBox lets the user type as well as pick, and a typed value that was never chosen
		// from the list leaves the selected key empty. Resolving the text against the same list
		// means typing BLK-001 in full works exactly as picking it does.
		_chosenBlockId: function () {
			var box = this.byId("dlgBlock");
			var key = box.getSelectedKey();
			if (key) {
				return key;
			}
			var typed = (box.getValue() || "").trim().toLowerCase();
			if (!typed) {
				return "";
			}
			var match = (this.getView().getModel("lookups").getProperty("/blocksOnly") || [])
				.filter(function (b) {
					return b.display.toLowerCase() === typed || b.code.toLowerCase() === typed;
				})[0];
			return match ? String(match.id) : "";
		},

		_lineDialog: function () {
			if (!this._lineDialogPromise) {
				this._lineDialogPromise = Fragment.load({
					id: this.getView().getId(),
					name: "farm.area.dashboard.view.LineDialog",
					controller: this
				}).then(function (dialog) {
					this.getView().addDependent(dialog);
					return dialog;
				}.bind(this));
			}
			return this._lineDialogPromise;
		},

		onSaveLine: function () {
			var strip = this.byId("lineDialogError");
			strip.setVisible(false);

			var blockId = this._chosenBlockId();
			var varietyId = this.byId("dlgVariety").getSelectedKey();
			if (!blockId || !varietyId) {
				strip.setText(blockId
					? "Choose the variety this block will carry."
					: "Choose the block this line plans.");
				strip.setVisible(true);
				return;
			}

			var harvest = this.byId("dlgHarvest").getValue();
			var body = {
				blockId: Number(blockId),
				caneVarietyId: Number(varietyId),
				plantingType: this.byId("dlgPlantingType").getSelectedKey(),
				projectedPlantingAreaHa: Number(this.byId("dlgArea").getValue()),
				plannedPlantingStart: this.byId("dlgLineStart").getValue(),
				plannedPlantingEnd: this.byId("dlgLineEnd").getValue(),
				expectedHarvestDate: harvest || null,
				priority: Number(this.byId("dlgPriority").getSelectedKey()),
				remark: this.byId("dlgLineRemark").getValue() || null
			};

			var path = "projections/" + this._id + "/lines";
			var request = this._editingLineId
				? this._api().put(path + "/" + this._editingLineId, body)
				: this._api().post(path, body);

			request.then(function (projection) {
				this.byId("lineDialog").close();
				MessageToast.show("Saved. The plan now covers " +
					formatter.hectares(projection.totalProjectedAreaHa) + " ha.");
				this._apply(projection);
			}.bind(this)).catch(function (error) {
				strip.setText(messageOf(error));
				strip.setVisible(true);
			});
		},

		onCloseLineDialog: function () {
			this.byId("lineDialog").close();
		},

		onDeleteLine: function (event) {
			var line = event.getSource().getBindingContext("p").getObject();
			MessageBox.confirm("Remove block " + line.blockCode + " from this plan?", {
				onClose: function (choice) {
					if (choice !== MessageBox.Action.OK) {
						return;
					}
					this._api().del("projections/" + this._id + "/lines/" + line.id)
						.then(function (projection) {
							MessageToast.show("Removed " + line.blockCode + ".");
							this._apply(projection);
						}.bind(this)).catch(this._showError.bind(this));
				}.bind(this)
			});
		},

		onOpenMap: function (event) {
			var line = event.getSource().getBindingContext("p").getObject();
			if (line.mapUrl) {
				window.open(line.mapUrl, "_blank", "noopener");
			}
		},

		// ---------------------------------------------------------------- chrome

		_showError: function (error) {
			var strip = this.byId("errorStrip");
			strip.setText(messageOf(error));
			strip.setVisible(true);
		},

		onBack: function () {
			this.getOwnerComponent().getRouter().navTo("projections");
		},

		onSignOut: function () {
			this._api().signOut();
			this._loaded = false;
			this.getOwnerComponent().getRouter().navTo("login");
		}
	});

	function confirmationFor(action) {
		switch (action) {
			case "Approve":
				return "Approving commits these blocks for the planned window. " +
					"No other plan will be able to take them. Approve?";
			case "Close":
				return "Closing finishes this plan. Nothing further can be done with it. Close it?";
			default:
				return "Continue?";
		}
	}

	function reasonPromptFor(action) {
		switch (action) {
			case "Reject":
				return "Why is this plan being rejected?";
			case "ReturnForCorrection":
				return "What has to change before this can be approved?";
			default:
				return "Why is this plan being revised? A new version will be opened carrying a copy " +
					"of the blocks, and this one will be marked superseded.";
		}
	}

	function messageOf(error) {
		var message = (error && error.message) || "The request failed.";
		if (error && error.fields && error.fields.length) {
			message += " " + error.fields.map(function (f) { return f.message; }).join(" ");
		}
		return message;
	}
});
