/**
@license
Copyright (c) 2024 trading_peter
This program is available under Apache License Version 2.0
*/

import '@tp/tp-icon/tp-icon.js';
import { formatTs } from '../helpers/time.js';
import { LitElement, html, css } from 'lit';
import icons from '../icons.js';
import { clipboard } from '@tp/helpers/clipboard.js';
import { shorten } from '../helpers/misc.js';

class ConversionCard extends clipboard(LitElement) {
  static get styles() {
    return [
      css`
        :host {
          display: block;
          width: 100%;
          padding: 15px 0;
        }

        :host([selected]) .wrap {
          border: solid 2px #ffffff;
        }

        .wrap {
          display: flex;
          flex-direction: column;
          background-color: #031331;
          border-radius: 10px;
          border: solid 2px transparent;
        }

        .header {
          display: flex;
          flex-direction: row;
          align-items: center;
          justify-content: space-between;
          padding: 10px 20px;
          border-radius: 10px 10px 0 0;
          background-color: #0c2553;
        }

        .header tp-icon {
          --tp-icon-width: 18px;
          --tp-icon-height: 18px;
        }

        .details {
          display: grid;
          grid-template-rows: auto 1fr;
          grid-template-columns: auto 1fr;
          grid-column-gap: 20px;
          padding: 10px 20px;
        }

        .flex {
          display: flex;
          flex-direction: row;
          align-items: center;
          justify-content: space-between;
        }

        .title {
          font-size: 20px;
        }

        .line {
          padding: 10px 0;
        }

        .error {
          background: rgb(92, 14, 14);
          font-weight: bold;
          color: #ffffff;
          padding: 5px;
          margin-top: 10px;
          border-radius: 4px;
        }
      `
    ];
  }

  render() {
    const { entry } = this;

    const fees = [];

    if (entry.Fee != '0' && entry.FeeCurrency) {
      fees.push(`-${entry.Fee} ${entry.FeeCurrency}`);
    }
    
    if (entry.QuoteFee != '0') {
      fees.push(`-${entry.QuoteFee} ${entry.QuoteFeeCurrency}`);
    }

    const feeStr = fees.join(' / ');

    return html`
      <div class="wrap">
        <div class="header">
          ${entry.RecID ? html`
          <div><tp-icon tooltip="Copy Trade ID" .icon=${icons.copy} @click=${() => this.copy(entry.RecID)}></tp-icon> ${shorten(entry.RecID)}</div>
          ` : null}
          <div>${entry.Account}</div>
          <div>${formatTs(entry.Ts)}</div>
        </div>
        <div class="details">
          <div></div>
          <div class="flex title">
            <div>
              ${entry.FromAmount} ${entry.From}
            </div>
            <div>
              <tp-icon .icon=${icons['arrow-right']}></tp-icon>
            </div>
            <div>
              ${entry.ToAmount} ${entry.To}
            </div>
          </div>

          <div class="line"><label>Purchase Costs</label></div>
          <div class="line flex">
            <div>
              ${entry.Result.CostEur}€
            </div>
            <div>
              ${entry.ToAmountEur}€
            </div>
          </div>

          <div class="line"><label>Fees</label></div>
          <div class="line flex">
            <div>
              ${feeStr}
            </div>
            <div>
              EUR: -${entry.Result.FeePayedEur}€
            </div>
          </div>
          
          <div class="line"><label>Properties</label></div>
          <div class="line flex">
            <div></div>
            <div>
              ${this.getProps(entry) || 'N/A'}
            </div>
          </div>
        </div>
        ${entry.Error ? html`
          <div class="error">${entry.Error}</div>
        ` : null}
      </div>
    `;
  }

  static get properties() {
    return {
      entry: { type: Object },
    };
  }

  getProps(entry) {
    const props = [
      entry.IsDerivative ? `Derivative` : null,
      entry.IsMarginTrade ? `Margin` : null,
      entry.IsPhysical ? `Physical` : null,
    ];

    return props.filter(i => i !== null).join(' / ');
  }
}

window.customElements.define('conversion-card', ConversionCard);