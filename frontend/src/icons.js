import { svg } from 'lit';

export default {
  'arrow-right': svg`
    <path fill="var(--tp-icon-color)" d="M4,15V9H12V4.16L19.84,12L12,19.84V15H4Z" />
  `,
  'copy': svg`
    <path fill="var(--tp-icon-color)" d="M19,21H8V7H19M19,5H8A2,2 0 0,0 6,7V21A2,2 0 0,0 8,23H19A2,2 0 0,0 21,21V7A2,2 0 0,0 19,5M16,1H4A2,2 0 0,0 2,3V17H4V3H16V1Z" />
  `,
  'down': svg`
    <path fill="var(--tp-icon-color)" d="M12,2A10,10 0 0,1 22,12A10,10 0 0,1 12,22A10,10 0 0,1 2,12A10,10 0 0,1 12,2M12,17L17,12H14V8H10V12H7L12,17Z" />
  `,
  'up': svg`
    <path fill="var(--tp-icon-color)" d="M12,22A10,10 0 0,1 2,12A10,10 0 0,1 12,2A10,10 0 0,1 22,12A10,10 0 0,1 12,22M12,7L7,12H10V16H14V12H17L12,7Z" />
  `,
  'alert': svg`
    <path fill="var(--tp-icon-color)" d="M13,13H11V7H13M13,17H11V15H13M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2Z" />
  `
};
