// js/components/notification.js
window.NotificationComponent = {
    props: {
        notification: Object
    },
    template: `
        <div v-if="notification.show" class="notification" :class="[notification.type, { show: notification.show }]">
            {{ notification.message }}
        </div>
    `
};
