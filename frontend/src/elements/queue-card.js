/**
@license
Copyright (c) 2024 trading_peter
This program is available under Apache License Version 2.0
*/

import { formatTs } from '../helpers/time.js';
import { LitElement, html, css } from 'lit';

class QueueCard extends LitElement {
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

        .details {
          padding: 10px 20px;
        }

        .queue-rec {
          display: grid;
          grid-template-columns: repeat(4, 1fr);
        }
      `
    ];
  }

  render() {
    const { asset } = this;

    return html`
      <div class="wrap">
        <div class="header">
          <div>${asset.Name}</div>
          <div>${asset.Total}</div>
        </div>
        <div class="details">
          <div class="queue-rec content">
            <div>
              <label>Units</label>
            </div>
            <div>
              <label>Cost per unit</label>
            </div>
            <div>
              <label>Fee per unit</label>
            </div>
            <div>
              <label>Acquired</label>
            </div>
          </div>
          ${asset.Entries.filter(entry => entry.UnitsLeft > 0).map(entry => html`
          <div class="queue-rec content">
            <div>
              <div>${entry.UnitsLeft} of ${entry.Units}</div>
            </div>
            <div>
              <div>${entry.UnitCostEur}€</div>
            </div>
            <div>
              <div>${entry.UnitFeeCostEur}€</div>
            </div>
            <div>
              <div>${formatTs(entry.Ts)}</div>
            </div>
          </div>
          `)}
        </div>
      </div>
    `;
  }

  static get properties() {
    return {
      asset: { type: Object },
    };
  }


}

window.customElements.define('queue-card', QueueCard);