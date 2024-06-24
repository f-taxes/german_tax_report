/**
@license
Copyright (c) 2022 trading_peter
This program is available under Apache License Version 2.0
*/

import '@tp/tp-button/tp-button.js';
import '@tp/tp-form/tp-form.js';
import '@tp/tp-dropdown/tp-dropdown.js';
import '@tp/tp-input/tp-input.js';
import '@lit-labs/virtualizer';
import './elements/conversion-card.js';
import './elements/queue-card.js';
import './elements/transfer-card.js';
import shared from './styles/shared.js';
import { LitElement, html, css } from 'lit';
import { fetchMixin } from '@tp/helpers/fetch-mixin.js';
import { DomQuery } from '@tp/helpers/dom-query.js';
import { closest } from '@tp/helpers/closest.js';
import icons from './icons.js';

class TheReport extends fetchMixin(DomQuery(LitElement)) {
  static get styles() {
    return [
      shared,
      css`
        :host {
          display: flex;
          flex-direction: column;
          padding: 20px;
          flex: 1;
        }

        .page {
          display: grid;
          grid-template-columns: 1fr 1fr;
          grid-column-gap: 20px;
          flex: 1;
        }

        .report-selector {
          display: flex;
          flex-direction: row;
          justify-content: space-between;
        }

        .report-selector tp-button {
          margin-left: 20px;
        }

        .report {
          display: flex;
          flex: 1;
          padding-top: 20px;
        }

        #virtualList {
          max-width: 640px;
          margin: 0 auto;
        }

        #virtualList,
        .empty-message {
          flex: 1;
        }

        conversion-card,
        transfer-card {
          padding-right: 20px;
        }

        .error-icon {
          --tp-icon-color: rgb(211, 33, 33);
        }
        
        .warning-icon {
          --tp-icon-color: rgb(207, 167, 6);
        }

        .marked-recs {
          display: flex;
          align-items: center;
        }

        .marked-recs tp-icon {
          margin-right: 10px;
        }

        .marked-recs + .marked-recs {
          margin-top: 10px;
        }
      `
    ];
  }

  render() {
    const { items, selIdx, selItem, errorCount, warningCount } = this;

    return html`
      <h2>Create a tax report</h2>
      <div class="report-selector">
        <tp-form @submit=${this.generate}>
          <form>
            <tp-dropdown name="year" .items=${this.listOfYears()} .default=${new Date().getFullYear()}></tp-dropdown>
            <tp-button submit>Generate</tp-button>
          </form>
        </tp-form>
        <div>
          <div class="marked-recs">
            <tp-icon .icon=${icons.alert} class="error-icon"></tp-icon>
            <div>
              ${errorCount} Errors
            </div>
            <div>
              <tp-icon .icon=${icons.down} @click=${this.nextError}></tp-icon>
              <tp-icon .icon=${icons.up} @click=${this.prevError}></tp-icon>
            </div>
          </div>
          <div class="marked-recs">
            <tp-icon .icon=${icons.alert} class="warning-icon"></tp-icon>
            <div>
              ${warningCount} Warnings
            </div>
          </div>
        </div>
      </div>

      <div class="page">
        <div class="report" @click=${this.itemClick}>
          <lit-virtualizer id="virtualList" part="list" scroller .items=${items} .renderItem=${(item, idx) => this.renderItem(item, idx, selIdx === idx)}></lit-virtualizer>
        </div>
        ${selItem ? html`
          <div class="details">
            ${Array.isArray(selItem.QueueBefore) && selItem.QueueBefore.filter(entry => entry.Name !== '').length > 0 ? html`
            <div>
              Queue Before:
              ${selItem.QueueBefore.map(asset => this.renderQueueRecs(asset))}
            </div>
            ` : null}
            ${Array.isArray(selItem.FromEntries) ? html`
            <div>
              Extracted Queue Entries:
              ${selItem.FromEntries.map(asset => this.renderQueueRecs(asset))}
            </div>
            ` : null}
            <div>
              Queue After:
              ${selItem.QueueAfter.map(asset => this.renderQueueRecs(asset))}
            </div>
          </div>
        ` : null}
      </div>
    `;
  }

  renderItem(item, idx, selected) {
    if (item.Type === 'conversion') {
      return html`<conversion-card .itemIdx=${idx} ?selected=${selected} .entry=${item}></conversion-card>`;
    }

    if (item.Type === 'deposit' || item.Type === 'withdrawal') {
      return html`<transfer-card .itemIdx=${idx} ?selected=${selected} .entry=${item}></transfer-card>`;
    }
  }

  renderQueueRecs(asset) {
    return html`<queue-card .asset=${asset}></queue-card>`;
  }

  static get properties() {
    return {
      items: { type: Array },
      selIdx: { type: Number },
      selItem: { type: Object },
      errors: { type: Array },
      errorCount: { type: Number },
      warningCount: { type: Number },
    };
  }

  constructor() {
    super();
    this.reset();
  }

  listOfYears() {
    const year = new Date().getFullYear();
    const years = [];

    for (let i = year - 10; i <= year; i++) {
      years.push({ label: i, value: i });
    }

    return years.reverse();
  }

  async generate(e) {
    this.reset();

    const btn = e.target.submitButton;
    btn.showSpinner();
    const resp = await this.post('/report/generate', { year: parseInt(e.detail.year, 10) });

    if (resp.result) {
      btn.showSuccess();
      this.items = resp.data;

      for (const item of resp.data) {
        if (item.Error) {
          this.errorCount++;
          if (item.RecID) {
            this.errors.push(item.RecID);
          }
        }
      }
    } else {
      btn.showError();
    }
  }

  reset() {
    this.items = [];
    this.errors = [];
    this.errorCount = 0;
    this.warningCount = 0;
    this.selError = -1;
    this.selWarning = -1;
  }

  itemClick(e) {
    const itemEl = closest(e.target, 'conversion-card') || closest(e.target, 'transfer-card');
    if (!itemEl) return;

    this.selIdx = itemEl.itemIdx;
    this.selItem = this.items[itemEl.itemIdx];
  }

  nextError() {
    if (this.errorCount == 0) return;

    this.selError = this.selError + 1;
    if (this.selError >= this.errors.length) {
      this.selError = 0;
    }

    const idx = this.items.findIndex(item => item.RecID === this.errors[this.selError]);
    if (idx > -1) {
      this.$.virtualList.element(idx)?.scrollIntoView({ block: 'start' });
    }
  }

  prevError() {
    if (this.errorCount == 0) return;

    this.selError = this.selError - 1;
    if (this.selError < 0) {
      this.selError = this.errors.length;
    }

    const idx = this.items.findIndex(item => item.RecID === this.errors[this.selError]);
    if (idx > -1) {
      this.$.virtualList.element(idx)?.scrollIntoView({ block: 'start' });
    }
  }
}

window.customElements.define('the-report', TheReport);