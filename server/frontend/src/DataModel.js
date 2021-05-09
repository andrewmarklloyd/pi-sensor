function translateStatus(status) {
    return {
        color: status === "OPEN" ? "red" : "green",
        icon: status === "OPEN" ? "unlock" : "lock"
    }
}

export default translateStatus;