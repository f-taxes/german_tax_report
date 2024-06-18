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

class TransferCard extends clipboard(LitElement) {
  static get styles() {
    return [
      css`
        :host {
          display: block;
          width: 100%;
          padding: 15px 0;
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

        .warning {
          background: rgb(221, 136, 7);
          font-weight: bold;
          color: #3f2300;
          padding: 5px;
          margin-top: 10px;
          border-radius: 4px;
        }
      `
    ];
  }

  render() {
    const { entry } = this;

    return html`
      <div class="wrap">
        <div class="header">
          ${entry.RecID ? html`
            <div><tp-icon tooltip="Copy Trade ID" .icon=${icons.copy} @click=${() => this.copy(entry.RecID)}></tp-icon> ${shorten(entry.RecID)}</div>
          ` : null}
          <div>${entry.Type[0].toUpperCase() + entry.Type.slice(1)}</div>
          <div>${formatTs(entry.Ts)}</div>
        </div>
        <div class="details">
          <div></div>
          <div class="flex title">
            <div>
              ${entry.Type === 'deposit' ? entry.Source || '[Unknown Source]' : entry.Account}
            </div>
            <div>
              <tp-icon .icon=${icons['arrow-right']}></tp-icon>
            </div>
            <div>
              ${entry.Type === 'deposit' ? entry.Account : entry.Destination  || '[Unknown Destination]'}
            </div>
          </div>

          <div></div>
          <div class="flex title">
            <div>
              ${entry.Type === 'deposit' ? '' : `${entry.Amount} ${entry.Asset}`}
            </div>
            <div></div>
            <div>
              ${entry.Type === 'deposit' ? `${entry.Amount} ${entry.Asset}` : ''}
            </div>
          </div>

          <div class="line"><label>Fees</label></div>
          <div class="line flex">
            <div>
              ${entry.Fee !== '0' && entry.Asset !== 'EUR' ? html`
                -${entry.Fee} ${entry.Asset}
              ` : null}
            </div>
            <div>
              EUR: ${entry.FeeEur === '0' ? entry.FeeEur : html`-${entry.FeeEur}`}€
            </div>
          </div>
        </div>
        ${entry.Warning ? html`
          <div class="warning">${entry.Warning}</div>
        ` : null}
      </div>
    `;
  }

  static get properties() {
    return { };
  }


}

window.customElements.define('transfer-card', TransferCard);